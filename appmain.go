package main

import (
	"fmt"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

const (
	APP_ICON_PATH          = "./img/appIcon.ico"
	TRAY_ICON_OK_PATH      = "./img/trayIconOK.ico"
	TRAY_ICON_ERR_PATH     = "./img/trayIconError.ico"
	TRAY_ICON_LOADING_PATH = "./img/trayIconLoading.ico"
)

type AppStatusType int

const (
	APP_STATUS_OK                AppStatusType = 0
	APP_STATUS_UNAUTHORIZED                    = 1
	APP_STATUS_BAD_CASERVER                    = 2
	APP_STATUS_BAD_PAGEANT_PROXY               = 4
	APP_STATUS_BAD_OPENSSH_AGENT               = 8
)

type UIAppType struct {
	mainWindow                *walk.MainWindow
	trayIcon                  *walk.NotifyIcon
	dashboardDlg, settingsDlg *walk.Dialog

	stepCaHealthLabel       *walk.Label
	pageantProxyHealthLabel *walk.Label
	opensshHealthLabel      *walk.Label
	userCertHealthLabel     *walk.Label

	authBtn *walk.PushButton

	userNameTxt *walk.LineEdit

	stepCaHealthOk bool
	pageantProxyOk bool
	opensshRunning bool
	userCertOk     bool
}

var (
	App                 *UIAppType = &UIAppType{}
	AppIcon             *walk.Icon
	TrayIconOKIcon      *walk.Icon
	TrayIconErrorIcon   *walk.Icon
	TrayIconLoadingIcon *walk.Icon

	refreshCaCheck      chan bool = make(chan bool)
	refreshCertCheck    chan bool = make(chan bool)
	refreshOpenSSHCheck chan bool = make(chan bool)
	refreshPageantCheck chan bool = make(chan bool)
)

const (
	CA_HEALTH_CHECK_DURATION     = 2 * 60 * time.Second
	USER_CERT_CHECK_DURATION     = 2 * 60 * time.Second
	OPENSSH_CHECK_DURATION       = 2 * 60 * time.Second
	PAGEANT_PROXY_CHECK_DURATION = 2 * 60 * time.Second
)

func (app *UIAppType) Init() {
	walk.AppendToWalkInit(func() {
		walk.FocusEffect, _ = walk.NewBorderGlowEffect(walk.RGB(0, 63, 255))
		walk.InteractionEffect, _ = walk.NewDropShadowEffect(walk.RGB(63, 63, 63))
		walk.ValidationErrorEffect, _ = walk.NewBorderGlowEffect(walk.RGB(255, 0, 0))
	})

	var err error
	if app.mainWindow, err = walk.NewMainWindowWithName(APP_NAME); err != nil {
		Logger.Panic("Failed to initilaize application main windows. Error: %v", err)
	}

	if AppIcon, err = walk.Resources.Icon(APP_ICON_PATH); err != nil {
		Logger.Panic("Failed to get resource '%v'. Error %v", APP_ICON_PATH, err)
	}

	if err = app.mainWindow.SetIcon(AppIcon); err != nil {
		Logger.Panic("Failed to initialize application icon. Error %v", err)
	}

	// BEGIN: INIT TRAY ICON //
	if TrayIconOKIcon, err = walk.Resources.Icon(TRAY_ICON_OK_PATH); err != nil {
		Logger.Panic("Failed to get resource '%v'. Error %v", TRAY_ICON_OK_PATH, err)
	}

	if TrayIconErrorIcon, err = walk.Resources.Icon(TRAY_ICON_ERR_PATH); err != nil {
		Logger.Panic("Failed to get resource '%v'. Error %v", TRAY_ICON_OK_PATH, err)
	}

	if TrayIconLoadingIcon, err = walk.Resources.Icon(TRAY_ICON_LOADING_PATH); err != nil {
		Logger.Panic("Failed to get resource '%v'. Error %v", TRAY_ICON_LOADING_PATH, err)
	}

	if app.trayIcon, err = walk.NewNotifyIcon(app.mainWindow); err != nil {
		Logger.Panic("Failed to initialize application tray icon. Error: %v", err)
	}

	if err = app.SetTrayIcon(TrayIconLoadingIcon); err != nil {
		Logger.Panic("Failed to initialize application tray icon. Error %v", err)
	}

	openDashboardAction := walk.NewAction()
	if err = openDashboardAction.SetText("Dashboard"); err != nil {
		Logger.Panic("Failed to initialize application tray icon. Error %v", err)
	}

	if err = app.trayIcon.ContextMenu().Actions().Add(openDashboardAction); err != nil {
		Logger.Panic("Failed to initialize application tray icon. Error %v", err)
	}

	openDashboardAction.Triggered().Attach(func() {
		app.OpenDashboardDialog()
	})

	if err = app.trayIcon.ContextMenu().Actions().Add(walk.NewSeparatorAction()); err != nil {
		Logger.Panic("Failed to initialize application tray icon. Error %v", err)
	}

	configStepCli := walk.NewAction()
	if err = configStepCli.SetText("Config StepCli"); err != nil {
		Logger.Panic("Failed to initialize application tray icon. Error %v", err)
	}

	if err = app.trayIcon.ContextMenu().Actions().Add(configStepCli); err != nil {
		Logger.Panic("Failed to initialize application tray icon. Error %v", err)
	}

	configStepCli.Triggered().Attach(func() {
		dlgCmd, err := app.OpenStepConfigDialog()
		if err != nil {
			Logger.Error("There was error with config step cli dialog. Error: %v", err)
		} else if dlgCmd == walk.DlgCmdOK {
			app.PushInfoNoti("New settings have been saved.")
		}
	})

	if err = app.trayIcon.ContextMenu().Actions().Add(walk.NewSeparatorAction()); err != nil {
		Logger.Panic("Failed to initialize application tray icon. Error %v", err)
	}

	openHomeDirAction := walk.NewAction()
	if err = openHomeDirAction.SetText("Home Folder"); err != nil {
		Logger.Panic("Failed to initialize application tray icon. Error %v", err)
	}

	if err = app.trayIcon.ContextMenu().Actions().Add(openHomeDirAction); err != nil {
		Logger.Panic("Failed to initialize application tray icon. Error %v", err)
	}

	openHomeDirAction.Triggered().Attach(func() {
		OpenHomeDir()
	})

	if err = app.trayIcon.ContextMenu().Actions().Add(walk.NewSeparatorAction()); err != nil {
		Logger.Panic("Failed to initialize application tray icon. Error %v", err)
	}

	refreshPageantProxyAction := walk.NewAction()
	if err = refreshPageantProxyAction.SetText("Refresh Pageant Proxy"); err != nil {
		Logger.Panic("Failed to initialize application tray icon. Error %v", err)
	}

	if err = app.trayIcon.ContextMenu().Actions().Add(refreshPageantProxyAction); err != nil {
		Logger.Panic("Failed to initialize application tray icon. Error %v", err)
	}

	refreshPageantProxyAction.Triggered().Attach(func() {
		PageantProxy.SendRestartSignal()
	})

	if err = app.trayIcon.ContextMenu().Actions().Add(walk.NewSeparatorAction()); err != nil {
		Logger.Panic("Failed to initialize application tray icon. Error %v", err)
	}

	exitAction := walk.NewAction()
	if err = exitAction.SetText("Exit"); err != nil {
		Logger.Panic("Failed to initialize application tray icon. Error %v", err)
	}

	if err = app.trayIcon.ContextMenu().Actions().Add(exitAction); err != nil {
		Logger.Panic("Failed to initialize application tray icon. Error %v", err)
	}

	exitAction.Triggered().Attach(func() {
		Configs.StoreConfigs()
		app.CleanUp()
		walk.App().Exit(0)
	})

	if err = app.trayIcon.SetToolTip("WinSSH Pageant Proxy initializing....!"); err != nil {
		Logger.Panic("Failed to set tray icon tool tip. Error: %v", err)
	}

	if err = app.trayIcon.SetVisible(true); err != nil {
		Logger.Panic("Failed to render tray icon. Error: %v", err)
	}
	// END: INIT TRAY ICON //
}

func (app *UIAppType) Start() {
	if !app.CheckStartupCondition() {
		Logger.Error("Startup condition was not satisfied. Exiting...")
		return
	}

	stepErr := StepCli.Init()
	if stepErr != nil {
		walk.MsgBox(app.mainWindow, APP_NAME+": Error", fmt.Sprintf("Failed to initialized Step CLI handler. Due to error: %s. Please check logs in %s for more detail", stepErr, APP_LOGS_DIR), walk.MsgBoxIconError|walk.MsgBoxOK)
		return
	}
	app.CheckStepCliConfiguration()

	go PageantProxy.Start()
	app.SetTrayIcon(TrayIconErrorIcon)
	app.trayIcon.SetToolTip("WinSSH Pageant Proxy")
	go app.CheckStatusLoop()
	app.mainWindow.Run()
	Logger.Error("AppMain: unexpected exit")
}

func (app *UIAppType) OpenDashboardDialog() (int, error) {
	var dlg *walk.Dialog
	var db *walk.DataBinder
	var refreshProvisionerBtn *walk.PushButton
	var proivisionerCombox *walk.ComboBox
	newConfigs := Configs
	err := Dialog{
		AssignTo: &dlg,
		Title:    fmt.Sprintf("%v: %v", APP_NAME, "Dashboard"),
		Icon:     AppIcon,
		DataBinder: DataBinder{
			AssignTo:       &db,
			Name:           "newConfigs",
			DataSource:     newConfigs,
			ErrorPresenter: ToolTipErrorPresenter{},
			AutoSubmit:     true,
			OnSubmitted: func() {
				Configs.UpdateConfig(*newConfigs)
				Configs.StoreConfigs()
			},
		},
		MinSize: Size{400, 250},
		MaxSize: Size{400, 250},
		Layout:  VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 5},
				Children: []Widget{
					Label{
						ColumnSpan: 1,
						Text:       "Provisoner: ",
					},
					ComboBox{
						ColumnSpan: 3,
						AssignTo:   &proivisionerCombox,
						Value:      Bind("StepDefaultProvisioner"),
						Model:      app.GetProvisoners(),
					},
					PushButton{
						AssignTo:   &refreshProvisionerBtn,
						Text:       "Refresh",
						ColumnSpan: 1,
						OnClicked: func() {
							StepCli.GetProvisionersSetWithRefreshing()
							proivisionerCombox.SetModel(app.GetProvisoners())
						},
					},
					Label{
						ColumnSpan: 1,
						Text:       "Username: ",
					},
					LineEdit{
						AssignTo:   &app.userNameTxt,
						ColumnSpan: 3,
						Text:       Bind("StepUsername", NonEmpty{}),
					},
					PushButton{
						AssignTo:   &app.authBtn,
						Text:       "Login",
						ColumnSpan: 1,
						OnClicked: func() {
							if app.authBtn.Text() == "Login" {
								err := StepCli.Login()
								if err != nil {
									Logger.Error("Failed to login user %v. Error: %v", Configs.StepUsername, err)
									app.PushErrNoti("Failed to login user %v. Error: %v", Configs.StepUsername, err)
									walk.MsgBox(app.mainWindow, APP_NAME+": Error", fmt.Sprintf("Failed to login user %v. Error: %v", Configs.StepUsername, err), walk.MsgBoxIconError|walk.MsgBoxOK)
								} else {
									app.PushInfoNoti("User %v logged in. Certificate has been updated.", Configs.StepUsername)
								}
							}

							if app.authBtn.Text() == "Logout" {
								err := StepCli.Logout()
								if err != nil {
									Logger.Error("Failed to logout user %v. Error: %v", Configs.StepUsername, err)
									app.PushErrNoti("Failed to logout user %v. Error: %v", Configs.StepUsername, err)
									walk.MsgBox(app.mainWindow, APP_NAME+": Error", fmt.Sprintf("Failed to logout user %v. Error: %v", Configs.StepUsername, err), walk.MsgBoxIconError|walk.MsgBoxOK)
								} else {
									app.PushInfoNoti("User %v logged out. Certificate of user has been removed.", Configs.StepUsername)
								}
							}
							refreshCertCheck <- true
						},
					},
				},
			},
			Composite{
				Layout: VBox{},
				Children: []Widget{
					Label{
						AssignTo:      &app.userCertHealthLabel,
						Text:          "<Checking>",
						TextColor:     walk.RGB(0, 0, 255),
						TextAlignment: AlignDefault,
						MinSize:       Size{350, 10},
						Background:    SolidColorBrush{walk.RGB(220, 220, 220)},
					},
					Label{
						AssignTo:      &app.stepCaHealthLabel,
						Text:          "<Checking>",
						TextColor:     walk.RGB(0, 0, 255),
						TextAlignment: AlignDefault,
						MinSize:       Size{350, 10},
						Background:    SolidColorBrush{walk.RGB(220, 220, 220)},
					},
					Label{
						AssignTo:      &app.pageantProxyHealthLabel,
						Text:          "<Checking>",
						TextColor:     walk.RGB(0, 0, 255),
						TextAlignment: AlignDefault,
						MinSize:       Size{350, 10},
						Background:    SolidColorBrush{walk.RGB(220, 220, 220)},
					},
					Label{
						AssignTo:      &app.opensshHealthLabel,
						Text:          "<Checking>",
						TextColor:     walk.RGB(0, 0, 255),
						TextAlignment: AlignDefault,
						MinSize:       Size{350, 10},
						Background:    SolidColorBrush{walk.RGB(220, 220, 220)},
					},
				},
			},
		},
	}.Create(nil)

	dlg.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		app.opensshHealthLabel = nil
		app.pageantProxyHealthLabel = nil
		app.stepCaHealthLabel = nil
		app.userCertHealthLabel = nil
	})

	return dlg.Run(), err
}

func (app *UIAppType) GetProvisoners() []string {
	provisionerSet, stepErr := StepCli.GetProvisionersSetNoRefresh()
	if stepErr != nil {
		output := [1]string{fmt.Sprintf("<Error: %v>", stepErr)}
		return output[:]
	}
	keys := make([]string, 0, len(provisionerSet.set))
	for k := range provisionerSet.set {
		keys = append(keys, k)
	}
	return keys
}

func (app *UIAppType) OpenStepConfigDialog() (int, error) {
	var dlg *walk.Dialog
	var db *walk.DataBinder
	var saveBtn, cancelBtn *walk.PushButton
	newConfigs := Configs

	return Dialog{
		AssignTo: &dlg,
		Icon:     AppIcon,
		Title:    fmt.Sprintf("%v: %v", APP_NAME, "Config StepCli"),
		DataBinder: DataBinder{
			AssignTo:       &db,
			Name:           "newConfigs",
			DataSource:     newConfigs,
			ErrorPresenter: ToolTipErrorPresenter{},
			OnSubmitted: func() {
				Configs.UpdateConfig(*newConfigs)
				Configs.StoreConfigs()
				stepErr := StepCli.ReConfigure()
				if stepErr != nil {
					walk.MsgBox(app.mainWindow, APP_NAME+": Error", fmt.Sprintf("Failed to reconfigure StepCli for your user. Error: %v!", stepErr), walk.MsgBoxIconError|walk.MsgBoxOK)
				} else {
					_, stepErr := StepCli.GetProvisionersSetWithRefreshing()
					if stepErr != nil {
						Logger.Error("Failed to refresh provisioner set after reconfiguring StepCli. Error: %v", stepErr)
						app.PushErrNoti("Failed to refresh provisioner set after reconfiguring StepCli. Error:s %v", stepErr)
					}
				}
				refreshCaCheck <- true
			},
		},
		MinSize: Size{400, 100},
		MaxSize: Size{400, 100},
		Layout:  VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 5},
				Children: []Widget{
					Label{
						ColumnSpan: 2,
						Text:       "Step Team Url: ",
					},
					LineEdit{
						ColumnSpan: 3,
						Text:       Bind("StepTeamUrl"),
					},
					Label{
						ColumnSpan: 2,
						Text:       "Step Team Name: ",
					},
					LineEdit{
						ColumnSpan: 3,
						Text:       Bind("StepTeamName", NonEmpty{}),
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &saveBtn,
						Text:     "Configure",
						OnClicked: func() {
							if err := db.Submit(); err != nil {
								Logger.Error("Failed to submit step config changes. Error: %v", err)
								app.PushErrNoti("Failed to submit step config changes. Error: %v", err)
								return
							}
							dlg.Accept()
						},
					},
					PushButton{
						AssignTo: &cancelBtn,
						Text:     "Cancel",
						OnClicked: func() {
							dlg.Cancel()
						},
					},
				},
			},
		},
	}.Run(nil)
}

func (app *UIAppType) CheckStatusLoop() {
	caHealthChan := app.CheckCaHealth()
	userCertHealthChan := app.CheckUserCertHealth()
	pageantProxyHealthChan := app.CheckPageantProxyHealth()
	openSSHHealthChan := app.CheckOpenSSHHealth()

	var appOk bool
	var caHealth, openSSH, userCert, pageantProxy string
	caHealthOk, openSSHOk, userCertOk, pageantProxyOk := false, false, false, false
	for {
		select {
		case caHealth = <-caHealthChan:
			caHealthOk = (caHealth == "OK")
		case openSSH = <-openSSHHealthChan:
			openSSHOk = openSSH == "OK"
		case userCert = <-userCertHealthChan:
			userCertOk = userCert == "OK"
		case pageantProxy = <-pageantProxyHealthChan:
			pageantProxyOk = pageantProxy == "OK"
		default:
			prevAppOk := appOk
			appOk = caHealthOk && userCertOk && openSSHOk && pageantProxyOk
			if appOk != prevAppOk {
				if !appOk {
					app.SetTrayIcon(TrayIconErrorIcon)
					//app.PushErrNoti(fmt.Sprintf(`WinSSH Pageant Proxy is not healthy.Error: CA = %s \| SSH = %s \| CERT = %s \| Proxy = %s`, caHealth, openSSH, userCert, pageantProxy))
				} else {
					app.SetTrayIcon(TrayIconOKIcon)
				}
			}

			if !caHealthOk {
				if app.stepCaHealthLabel != nil {
					lastText := app.stepCaHealthLabel.Text()
					newText := fmt.Sprintf("<Error>      %s", caHealth)
					if lastText != newText {
						app.stepCaHealthLabel.SetText(newText)
						app.stepCaHealthLabel.SetTextColor(walk.RGB(255, 0, 0))
					}
				}
			} else {
				if app.stepCaHealthLabel != nil {
					lastText := app.stepCaHealthLabel.Text()
					newText := "<OK>         CA Server is healthy"
					if lastText != newText {
						app.stepCaHealthLabel.SetText(newText)
						app.stepCaHealthLabel.SetTextColor(walk.RGB(0, 255, 0))
					}
				}
			}

			if !openSSHOk {
				if app.opensshHealthLabel != nil {
					lastText := app.opensshHealthLabel.Text()
					newText := fmt.Sprintf("<Error>      %s", openSSH)
					if lastText != newText {
						app.opensshHealthLabel.SetText(newText)
						app.opensshHealthLabel.SetTextColor(walk.RGB(255, 0, 0))
					}
				}
			} else {
				if app.opensshHealthLabel != nil {
					lastText := app.opensshHealthLabel.Text()
					newText := "<OK>         OpenSSH Agent is running"
					if lastText != newText {
						app.opensshHealthLabel.SetText(newText)
						app.opensshHealthLabel.SetTextColor(walk.RGB(0, 255, 0))
					}
				}
			}

			if !userCertOk {
				if app.userCertHealthLabel != nil {
					lastText := app.userCertHealthLabel.Text()
					newText := fmt.Sprintf("<Error>      %s", userCert)
					if lastText != newText {
						app.userCertHealthLabel.SetText(newText)
						app.userCertHealthLabel.SetTextColor(walk.RGB(255, 0, 0))
					}
				}

				if app.authBtn != nil && app.authBtn.Text() == "Logout" {
					app.authBtn.SetText("Login")
				}

				if app.userNameTxt != nil && !app.userNameTxt.Enabled() {
					app.userNameTxt.SetEnabled(true)
				}
			} else {
				if app.userCertHealthLabel != nil {
					lastText := app.userCertHealthLabel.Text()
					newText := "<OK>         User's certificate is valid"
					if lastText != newText {
						app.userCertHealthLabel.SetText(newText)
						app.userCertHealthLabel.SetTextColor(walk.RGB(0, 255, 0))
					}
				}

				if app.authBtn != nil && app.authBtn.Text() == "Login" {
					app.authBtn.SetText("Logout")
				}

				if app.userNameTxt != nil && app.userNameTxt.Enabled() {
					app.userNameTxt.SetEnabled(false)
				}
			}

			if !pageantProxyOk {
				if app.pageantProxyHealthLabel != nil {
					lastText := app.pageantProxyHealthLabel.Text()
					newText := fmt.Sprintf("<Error>      %s", pageantProxy)
					if lastText != newText {
						app.pageantProxyHealthLabel.SetText(newText)
						app.pageantProxyHealthLabel.SetTextColor(walk.RGB(255, 0, 0))
					}
				}
			} else {
				if app.pageantProxyHealthLabel != nil {
					lastText := app.pageantProxyHealthLabel.Text()
					newText := "<OK>         Pageant Proxy is healthy"
					if lastText != newText {
						app.pageantProxyHealthLabel.SetText(newText)
						app.pageantProxyHealthLabel.SetTextColor(walk.RGB(0, 255, 0))
					}
				}
			}

			time.Sleep(1 * time.Second)
		}
	}
}

func (app *UIAppType) CheckUserCertHealth() <-chan string {
	output := make(chan string)
	go func() {
		timeoutchan := make(chan bool)
		for {
			userCertOk, stepErr := StepCli.GetUserCertOk()
			if !userCertOk {
				output <- fmt.Sprintf("User's certificate invalid. Error: %v", stepErr)
			} else {
				output <- "OK"
			}

			go func() {
				<-time.After(USER_CERT_CHECK_DURATION)
				timeoutchan <- true
			}()

			select { // wait for sleep timeout or refresh signal
			case <-timeoutchan:
				break
			case <-refreshCertCheck:
				Logger.Info("User's certificate check refreshed")
				break
			}
		}
	}()
	return output
}

func (app *UIAppType) CheckCaHealth() <-chan string {
	output := make(chan string)
	go func() {
		timeoutchan := make(chan bool)
		for {
			stepCaHealthOk, stepErr := StepCli.GetCaHealth()
			if !stepCaHealthOk {
				output <- fmt.Sprintf("CA Health Check Failed. Error: %v", stepErr)
			} else {
				output <- "OK"
			}

			go func() {
				<-time.After(CA_HEALTH_CHECK_DURATION)
				timeoutchan <- true
			}()

			select { // wait for sleep timeout or refresh signal
			case <-timeoutchan:
				break
			case <-refreshCaCheck:
				Logger.Info("CA health check refreshed")
				break
			}
		}
	}()
	return output
}

func (app *UIAppType) CheckOpenSSHHealth() <-chan string {
	output := make(chan string)
	go func() {
		timeoutchan := make(chan bool)
		for {
			opensshAgentRunning := IsProcessNameExist("ssh-agent", false)
			if !opensshAgentRunning {
				output <- "OpenSSH Agent not running"
			} else {
				output <- "OK"
			}

			go func() {
				<-time.After(OPENSSH_CHECK_DURATION)
				timeoutchan <- true
			}()

			select { // wait for sleep timeout or refresh signal
			case <-timeoutchan:
				break
			case <-refreshOpenSSHCheck:
				Logger.Info("OpenSSH check refreshed")
				break
			}
		}
	}()
	return output
}

func (app *UIAppType) CheckPageantProxyHealth() <-chan string {
	output := make(chan string)
	go func() {
		timeoutchan := make(chan bool)
		for {
			pageantProxyOk := (PageantProxy.NamedPipe_OK && PageantProxy.WM_CopyData_OK)
			if !pageantProxyOk {
				errorMsg := ""
				if !PageantProxy.NamedPipe_OK {
					errorMsg = errorMsg + "| NamedPipe proxy has errors"
				}

				if !PageantProxy.WM_CopyData_OK {
					errorMsg = errorMsg + "| WM_CopyData proxy has errors"
				}
				output <- errorMsg
			} else {
				output <- "OK"
			}

			go func() {
				<-time.After(PAGEANT_PROXY_CHECK_DURATION)
				timeoutchan <- true
			}()

			select { // wait for sleep timeout or refresh signal
			case <-timeoutchan:
				break
			case <-refreshPageantCheck:
				Logger.Info("Pageant Proxy check refreshed")
				break
			}
		}
	}()
	return output
}

func (app *UIAppType) SetTrayIcon(icon *walk.Icon) error {
	return app.trayIcon.SetIcon(icon)
}

func (app *UIAppType) ErrorMsgBox(message string) {
	app.MsgBox("ERROR", message)
}

func (app *UIAppType) WarningMsgBox(message string) {
	app.MsgBox("WARNING", message)
}

func (app *UIAppType) MsgBox(messagetype string, message string) {
	var style walk.MsgBoxStyle
	if messagetype == "ERROR" {
		style = walk.MsgBoxIconError | walk.MsgBoxOK
	}

	if messagetype == "WARNING" {
		style = walk.MsgBoxIconWarning | walk.MsgBoxOK
	}
	walk.MsgBox(app.mainWindow, APP_NAME+": "+messagetype, message, style)
}

func (app *UIAppType) PushInfoNoti(format string, v ...interface{}) {
	app.PushNoti("INFO", format, v...)
}

func (app *UIAppType) PushWarnNoti(format string, v ...interface{}) {
	app.PushNoti("WARNING", format, v...)
}

func (app *UIAppType) PushErrNoti(format string, v ...interface{}) {
	app.PushNoti("ERROR", format, v...)
}

func (app *UIAppType) PushNoti(msgtype string, format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	if msgtype == "INFO" {
		go app.trayIcon.ShowInfo(APP_NAME, message)
	}

	if msgtype == "WARNING" {
		go app.trayIcon.ShowWarning(APP_NAME, message)
	}

	if msgtype == "ERROR" {
		go app.trayIcon.ShowError(APP_NAME, message)
	}
}

func (app *UIAppType) CheckStartupCondition() bool {
	var ok bool
	ok = !app.IsPageantProcessRunning()
	ok = ok && !app.IsAnotherMeRunning()
	return ok
}

func (app *UIAppType) IsPageantProcessRunning() bool {
	isPageantRunning := IsProcessNameExist("pageant", true)
	if isPageantRunning {
		answer := walk.MsgBox(app.mainWindow, APP_NAME+": Error", "Another Pageant process is running. Do you want to stop it ?", walk.MsgBoxIconError|walk.MsgBoxYesNo)
		if answer == win.IDNO {
			isPageantRunning = true
			walk.MsgBox(app.mainWindow, APP_NAME+": Info", "Please consider stoping the running Pageant later, and start this WinSSH Pageant Proxy again!", walk.MsgBoxIconExclamation|walk.MsgBoxOK)
		}

		if answer == win.IDYES {
			stopped := StopProcessWithName("pageant")
			if !stopped {
				isPageantRunning = true
				walk.MsgBox(app.mainWindow, APP_NAME+": Error", "Failed to stop 'pageant' process. Please stop it manually!", walk.MsgBoxIconError|walk.MsgBoxOK)
			} else {
				app.PushInfoNoti("Stopped running Pageant process. Starting up WinSSH Pageant Proxy...")
				isPageantRunning = false
			}
		}
	}
	return isPageantRunning
}

func (app *UIAppType) IsAnotherMeRunning() bool {
	pipeName, err := PageantProxy.GetPagentPipeName()
	if err != nil {
		Logger.Error("Failed to determine if another me is already running or not. Will starting up normally. Error: %v", err)
		return false
	}
	isPageantNamedPipeExists := IsFileExist(pipeName)
	if isPageantNamedPipeExists {
		walk.MsgBox(app.mainWindow, APP_NAME+": Error", "WinSSH Pageant Proxy is already running!", walk.MsgBoxIconError|walk.MsgBoxOK)
	}
	return isPageantNamedPipeExists
}

func (app *UIAppType) CheckStepCliConfiguration() {
	caHealthOk, stepErr := StepCli.GetCaHealth()
	if caHealthOk {
		_, stepErr := StepCli.GetProvisionersSetWithRefreshing()
		if stepErr != nil {
			Logger.Error("Failed to refresh provisioner set at start up. Error %v", stepErr)
			app.PushErrNoti("Faild to refresh provisioner set at start up. Error %v", stepErr)
		}
		return
	} else {
		if stepErr == STEPERR_STEPCA_NOT_CONFIGURED {
			walk.MsgBox(app.mainWindow, APP_NAME+": Error", "StepCli have not been configured for your user. Please configured it manually using 'Config StepCli' in tray menu!", walk.MsgBoxIconError|walk.MsgBoxOK)
			return
		} else {
			walk.MsgBox(app.mainWindow, APP_NAME+": Error", fmt.Sprintf("An error occured while checking for StepCli configuration. Error: %v", stepErr), walk.MsgBoxIconError|walk.MsgBoxOK)
			return
		}
	}
}

type NonEmptyValidatorType struct {
}

var NonEmptyValidatorSingleton walk.Validator = NonEmptyValidatorType{}

func NonEmptyValidator() walk.Validator {
	return NonEmptyValidatorSingleton
}

func (NonEmptyValidatorType) Validate(v interface{}) error {
	if v == nil {
		// For Widgets like ComboBox nil is passed to indicate "no selection".
		return walk.NewValidationError(
			"Field Required",
			"Please enter value for it")
	}

	return nil
}

type NonEmpty struct {
}

func (NonEmpty) Create() (walk.Validator, error) {
	return NonEmptyValidator(), nil
}

func (app *UIAppType) CleanUp() {
	Logger.Info("Cleaning up UI app resource")
	if app.dashboardDlg != nil {
		app.dashboardDlg.Dispose()
		app.dashboardDlg.Close(0)
	}

	if app.settingsDlg != nil {
		app.settingsDlg.Dispose()
		app.settingsDlg.Close(0)
	}

	if app.trayIcon != nil {
		app.trayIcon.ContextMenu().Dispose()
		app.trayIcon.Dispose()
	}

	if app.mainWindow != nil {
		app.mainWindow.Dispose()
		app.mainWindow.Close()
	}

}
