package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os/user"
	"time"

	//"runtime"
	"strings"
	"syscall"
	"unsafe"

	"encoding/binary"

	"github.com/Microsoft/go-winio"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

// copyDataStruct is used to pass data in the WM_COPYDATA message.
// We directly pass a pointer to our copyDataStruct type, be careful that it matches the Windows type exactly
type copyDataStruct struct {
	dwData uintptr
	cbData uint32
	lpData uintptr
}

type PageantProxyType struct {
	WM_CopyData_OK         bool
	NamedPipe_OK           bool
	NamedPipe_Listener     net.Listener
	NamedPipe_Connection   net.Conn
	winWHND                win.HWND
	isRestarting           bool
	proxyRestartChn        chan int
	NamedPipe_stoppedChn   chan int
	NamedPipe_stopChn      chan int
	WM_COPYDATA_stopChn    chan int
	WM_COPYDATA_stoppedChn chan int
}

var (
	PageantProxy *PageantProxyType = &PageantProxyType{
		WM_CopyData_OK:         true,
		NamedPipe_OK:           true,
		NamedPipe_Listener:     nil,
		NamedPipe_Connection:   nil,
		winWHND:                win.HWND(0),
		isRestarting:           false,
		NamedPipe_stopChn:      make(chan int),
		WM_COPYDATA_stopChn:    make(chan int),
		NamedPipe_stoppedChn:   make(chan int),
		WM_COPYDATA_stoppedChn: make(chan int),
	}
)

func (p *PageantProxyType) openFileMap(dwDesiredAccess uint32, bInheritHandle uint32, mapNamePtr uintptr) (windows.Handle, error) {
	mapPtr, _, err := PROC_OPENFILE_MAPPING_A.Call(uintptr(dwDesiredAccess), uintptr(bInheritHandle), mapNamePtr)
	if err != nil && err.Error() == "The operation completed successfully." {
		err = nil
	}

	if err != nil {
		Logger.Error("PageantProxy: Error openning file map. Error: %v", err)
		p.WM_CopyData_OK = false
	}

	return windows.Handle(mapPtr), err
}

func (p *PageantProxyType) wndProcCallBack(hWnd win.HWND, message uint32, wParam uintptr, lParam uintptr) uintptr {
	Logger.Info("PageantProxy: wndProcCallBack receive message %v", message)
	switch message {
	case win.WM_DESTROY:
		{
			Logger.Info("PageantProxy: receive WM_DESTROY messaage. Stopping message loop.")
			win.PostQuitMessage(0)
		}
	case 800:
		{
			Logger.Info("PageantProxy: Receive user lock message 800")
			p.SendRestartSignal()
			return 1
		}
	case win.WM_COPYDATA:
		{
			copyData := (*copyDataStruct)(unsafe.Pointer(lParam))

			fileMap, err := p.openFileMap(FILE_MAP_ALL_ACCESS, 0, copyData.lpData)
			if err != nil {
				Logger.Error("PageantProxy: Failed to open file map. Error: %v", err)
				p.WM_CopyData_OK = false
				return 0
			}
			defer windows.CloseHandle(fileMap)

			// check security
			ourself, err := GetUserSID()
			if err != nil {
				Logger.Error("PageantProxy: Failed to get UserSID. Error %v", err)
				p.WM_CopyData_OK = false
				return 0
			}
			ourself2, err := GetDefaultSID()
			if err != nil {
				Logger.Error("PageantProxy: Failed to get DefaultSID. Error %v", err)
				p.WM_CopyData_OK = false
				return 0
			}
			mapOwner, err := GetHandleSID(fileMap)
			if err != nil {
				Logger.Error("PageantProxy: Failed to get HandleSID. Error %v", err)
				p.WM_CopyData_OK = false
				return 0
			}
			if !windows.EqualSid(mapOwner, ourself) && !windows.EqualSid(mapOwner, ourself2) {
				Logger.Error("PageantProxy: file map is already own by something else")
				p.WM_CopyData_OK = false
				return 0
			}

			// Passed security checks, copy data
			sharedMemory, err := windows.MapViewOfFile(fileMap, 2, 0, 0, 0)
			if err != nil {
				Logger.Error("PageantProxy: Failed to get shared memory. Error: %v", err)
				p.WM_CopyData_OK = false
				return 0
			}
			defer windows.UnmapViewOfFile(sharedMemory)

			sharedMemoryArray := (*[AgentMaxMessageLength]byte)(unsafe.Pointer(sharedMemory))

			size := binary.BigEndian.Uint32(sharedMemoryArray[:4]) + 4
			// size += 4
			if size > AgentMaxMessageLength {
				Logger.Error("PageantProxy: Message size from file map is too large, size = %v", size)
				p.WM_CopyData_OK = false
				return 0
			}

			// result, err := sshagent.QueryAgent(*sshPipe, sharedMemoryArray[:size], sshagent.AgentMaxMessageLength)
			result, err := QueryAgent(SSH_AGENT_PIPE, sharedMemoryArray[:size])
			if err != nil {
				Logger.Error("PageantProxy: Failed to query for sshagent. Result: %v. Error: %v", result, err)
				p.WM_CopyData_OK = false
				return 0
			}
			copy(sharedMemoryArray[:], result)
			Logger.Info("PageantProxy: Successfully copied data from sshagent")
			p.WM_CopyData_OK = true
			return 1
		}
	}

	return win.DefWindowProc(hWnd, message, wParam, lParam)
}

func (p *PageantProxyType) registerPageantWindow(hInstance win.HINSTANCE) (atom win.ATOM) {
	var wc win.WNDCLASSEX
	wc.Style = 0

	wc.CbSize = uint32(unsafe.Sizeof(wc))
	wc.LpfnWndProc = syscall.NewCallback(p.wndProcCallBack)
	wc.CbClsExtra = 0
	wc.CbWndExtra = 0
	wc.HInstance = hInstance
	wc.HIcon = win.LoadIcon(0, win.MAKEINTRESOURCE(win.IDI_APPLICATION))
	wc.HCursor = win.LoadCursor(0, win.MAKEINTRESOURCE(win.IDC_IBEAM))
	wc.HbrBackground = win.GetSysColorBrush(win.BLACK_BRUSH)
	wc.LpszMenuName = nil
	wc.LpszClassName = syscall.StringToUTF16Ptr(WND_CLASSNAME)
	wc.HIconSm = win.LoadIcon(0, win.MAKEINTRESOURCE(win.IDI_APPLICATION))

	return win.RegisterClassEx(&wc)
}

func (p *PageantProxyType) Start_Pageant_WM_COPYDATA_Proxy() {
	Logger.Info("PageantProxy: Starting up Pageant WM_COPYDATA Proxy Server")
	inst := win.GetModuleHandle(nil)
	atom := p.registerPageantWindow(inst)
	if atom == 0 {
		Logger.Error("PageantProxy: WM_COPYDATA RegisterClass failed: %d", win.GetLastError())
		p.WM_CopyData_OK = false
	}

	// CreateWindowEx
	p.winWHND = win.CreateWindowEx(win.WS_EX_APPWINDOW,
		syscall.StringToUTF16Ptr(WND_CLASSNAME),
		syscall.StringToUTF16Ptr(WND_CLASSNAME),
		0,
		0, 0,
		0, 0,
		0,
		0,
		inst,
		nil)

	//runtime.LockOSThread()
	msg := (*win.MSG)(unsafe.Pointer(win.GlobalAlloc(0, unsafe.Sizeof(win.MSG{}))))
	defer win.GlobalFree(win.HGLOBAL(unsafe.Pointer(msg)))
	go func() {
		Logger.Info("PageantProxy: WM_COPYDATA event handler coroutine started")
		for win.GetMessage(msg, 0, 0, 0) > 0 {
			//if !win.IsDialogMessage(p.winWHND, msg) {
			//	win.TranslateMessage(msg)
			//	win.DispatchMessage(msg)
			//}
			//time.Sleep(100 * time.Millisecond)
		}
	}()

	close := false
	for !close {
		select {
		case <-p.WM_COPYDATA_stopChn:
			{
				Logger.Info("PageantProxy: Receive stop signal for WM_COPYDATA proxy. Now stopping!")
				p.Stop_Pageant_WM_COPYDATA_Proxy()
				close = true
			}
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
	Logger.Info("PageantProxy: sending WM_COPYDATA stopped signal back to main handler")
	p.WM_COPYDATA_stoppedChn <- 1
	Logger.Info("PageantProxy: finished sending WM_COPYDATA stopped signal back to main handler")
}

func (p *PageantProxyType) Stop_Pageant_WM_COPYDATA_Proxy() bool {
	var stopped bool
	var ok bool
	Logger.Info("PageantProxy: sending WM_CLOSE message to WM_COPYDATA message handler")
	ret := win.SendMessage(p.winWHND, win.WM_CLOSE, 0, 0)
	Logger.Info("PageantProxy: finished sending WM_CLOSE message. Result %v", ret)

	//Logger.Info("PageantProxy: checking and trying to cleanup WNDCLass resource of WM_COPYDATA proxy")
	//ok = win.DestroyWindow(p.winWHND)
	//if !ok {
	//	Logger.Error("PageantProxy: Failed to destroy window class for WM_COPYDATA proxy. Error: %v", win.GetLastError())
	//	stopped = false
	//}

	Logger.Info("PageantProxy: checking and trying to unregister WM_COPYDATA proxy WNDClass")
	ok = win.UnregisterClass(syscall.StringToUTF16Ptr(WND_CLASSNAME))
	if !ok {
		Logger.Error("PageantProxy: Failed to unregister window class for WM_COPYDATA proxy. Error: %v", win.GetLastError())
		stopped = false
	}

	Logger.Info("PageantProxy: finished closing WM_COPYDATA proxy resources")
	return stopped
}

///////////////////////////////////////

func (p *PageantProxyType) pipeListen(pageantConn net.Conn) {
	defer func() {
		if pageantConn != nil {
			pageantConn.Close()
		}
	}()
	reader := bufio.NewReader(pageantConn)

	for {
		lenBuf := make([]byte, 4)
		_, err := io.ReadFull(reader, lenBuf)
		if err != nil {
			p.NamedPipe_OK = false
			Logger.Error("PageantProxy: failed to read query data length from named pipe. Error: %v", err)
			return
		}

		bufferLen := binary.BigEndian.Uint32(lenBuf)
		readBuf := make([]byte, bufferLen)
		_, err = io.ReadFull(reader, readBuf)
		if err != nil {
			p.NamedPipe_OK = false
			Logger.Error("PageantProxy: failed to read query data from named pipe. Error: %v", err)
			return
		}

		result, err := QueryAgent(SSH_AGENT_PIPE, append(lenBuf, readBuf...))
		if err != nil {
			p.NamedPipe_OK = false
			Logger.Error("PageantProxy: failed to query from openssh-agent. Error: %v", err)
			return
		}

		_, err = pageantConn.Write(result)
		if err != nil {
			p.NamedPipe_OK = false
			Logger.Error("PageantProxy: failed to write result data to named pipe. Error: %v", err)
			return
		}
		p.NamedPipe_OK = true
		Logger.Info("PageantProxy: successfully write result data to named pipe. Result: %s", result)
	}
}

func (p *PageantProxyType) GetPagentPipeName() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		Logger.Error("PageantProxy: Failed to query current username from system. Error: %v", err)
		p.NamedPipe_OK = false
		return "", err
	}
	pipeName := fmt.Sprintf(AGENT_PIPE_NAME, strings.Split(currentUser.Username, `\`)[1], CapiObfuscateString(WND_CLASSNAME))
	return pipeName, nil
}

func (p *PageantProxyType) Start_PageantNamedPipeProxy() {
	Logger.Info("PageantProxy: Starting up NamedPipe Proxy Server")
	var err error
	pipeName, err := p.GetPagentPipeName()
	if err != nil {
		Logger.Error("PageantProxy: Failed to get name of named-pipe. Error: %v", err)
		p.NamedPipe_OK = false
		return
	}
	p.NamedPipe_Listener, err = winio.ListenPipe(pipeName, nil)
	if err != nil {
		Logger.Error("PageantProxy: Failed to create listener on named pipe %v. Error: %v", pipeName, err)
		p.NamedPipe_OK = false
		return
	}
	defer func() {
		if p.NamedPipe_Listener != nil {
			p.NamedPipe_Listener.Close()
		}
	}()

	stopped := make(chan int)
	go func() {
		Logger.Info("PageantProxy: NamedPipe proxy message handler coroutine started")
	out:
		for {
			select {
			case <-stopped:
				break out
			default:
				{
					p.NamedPipe_Connection, err = p.NamedPipe_Listener.Accept()
					if p.NamedPipe_Connection != nil {
						if err != nil {
							Logger.Error("PageantProxy: Failed to process message on named pipe. Error: %v", err)
							p.NamedPipe_OK = false
						} else {
							Logger.Info("PageantProxy: receive new message on NamedPipe Proxy.")
							go p.pipeListen(p.NamedPipe_Connection)
						}
					} else {
						Logger.Info("PageantProxy: NamedPipe '%v' connection is closed!", pipeName)
					}
				}
			}
		}
		Logger.Info("PageantProxy: NamedPipe proxy message handler coroutine stopped.")
	}()

	close := false
	for !close {
		select {
		case <-p.NamedPipe_stopChn:
			{
				Logger.Info("PageantProxy: Receive stop signal for NamedPipe proxy. Now stopping!")
				p.Stop_PageantNamedPipeProxy()
				stopped <- 1
				close = true
			}
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
	Logger.Info("PageantProxy: sending NamedPipe proxy stopped signal to main handler")
	p.NamedPipe_stoppedChn <- 1
	Logger.Info("PageantProxy: finsihed sending NamedPipe proxy stopped signal to main handler")
}

func (p *PageantProxyType) Stop_PageantNamedPipeProxy() bool {
	Logger.Info("PageantProxy: check and try to close NamedPipe proxy resource")
	if p.NamedPipe_Connection != nil {
		err := p.NamedPipe_Connection.Close()
		if err != nil {
			Logger.Error("PageantProxy: failed to close NamedPipe connection. Error %v", err)
		}
	}

	if p.NamedPipe_Listener != nil {
		err := p.NamedPipe_Listener.Close()
		if err != nil {
			Logger.Error("PageantProxy: failed to close NamedPipe listener. Error %v", err)
		}
	}
	Logger.Info("PageantProxy: finished closing NamedPipe proxy resource")
	return true
}

func (p *PageantProxyType) SendRestartSignal() {
	if p.proxyRestartChn != nil && !p.isRestarting {
		Logger.Info("PageantProxy: sending restart signal to main handler")
		p.isRestarting = true
		p.proxyRestartChn <- 1
	}
}

func (p *PageantProxyType) Start() bool {
	if p.proxyRestartChn == nil {
		p.proxyRestartChn = make(chan int)
	}
	go func() {
		for {
			select {
			case <-p.proxyRestartChn:
				Logger.Info("PageentProxy: receive restart signal. Restarting")
				go p.Restart()
			default:
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	go p.Start_PageantNamedPipeProxy()
	go p.Start_Pageant_WM_COPYDATA_Proxy()

	Logger.Info("PageantProxy: WM_COPYDATA proxy and NamedPipe proxy started")
	p.isRestarting = false
	return true
}

func (p *PageantProxyType) Stop() bool {
	Logger.Info("PageantProxy: sending stop signal to NamedPipe proxy coroutine")
	p.NamedPipe_stopChn <- 1
	<-p.NamedPipe_stoppedChn
	Logger.Info("PageantProxy: received signal from NamedPipe proxy that it is stopped")

	Logger.Info("PageantProxy: sending stop signal to WM_COPYDATA proxy coroutine")
	p.WM_COPYDATA_stopChn <- 1
	<-p.WM_COPYDATA_stoppedChn
	Logger.Info("PageantProxy: received signal from WM_COPYDATA proxy that it is stopped")

	return true
}

func (p *PageantProxyType) Restart() bool {
	stopped := p.Stop()
	if !stopped {
		Logger.Error("PageantProxy: proxies was not successfully stopped")
	} else {
		Logger.Info("PageantProxy: all proxies stopped. Now starting again.")
	}
	go p.Start()
	return true
}

func (p *PageantProxyType) IsHealthy() bool {
	return p.NamedPipe_OK && p.WM_CopyData_OK
}
