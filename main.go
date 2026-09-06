package main

import (
	"favapps/app"
	"fmt"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	tviewApp        *tview.Application
	list            *tview.List
	details         *tview.TextView
	cmdView         *tview.TextView
	flex            *tview.Flex
	pages           *tview.Pages
	apps            []app.App
	filteredIndices []int
	selectedIndices map[string]bool // Track multi-selected items by app name

	lastSearchText   string            // Store last search text for restoration
	searchBoxVisible bool              // Track if search box is currently visible
	searchInput      *tview.InputField // Reference to search input field

	titleColor          = "green"
	helpColor           = "blue"
	borderNormalColor   = tcell.ColorBlack
	borderSelectedColor = tcell.ColorGreen
)

func main() {
	var err error
	app.LoadConfig()

	apps, err = app.LoadApps()
	if err != nil {
		panic(err)
	}
	resetFilteredIndices()
	selectedIndices = make(map[string]bool)

	tviewApp = tview.NewApplication()
	pages = tview.NewPages()

	tview.Styles.BorderColor = borderNormalColor
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault

	list = tview.NewList().ShowSecondaryText(false).
		SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorGreen).Foreground(tcell.ColorDefault))
	list.SetBorder(true).SetTitle(fmt.Sprintf("[%s] Favourite Apps ", titleColor)).SetTitleAlign(tview.AlignLeft)
	list.SetFocusFunc(func() { list.SetBorderColor(borderSelectedColor) })
	list.SetBlurFunc(func() { list.SetBorderColor(borderNormalColor) })

	details = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)
	details.SetBorder(true).SetTitle(fmt.Sprintf("[%s] Description ", titleColor)).SetTitleAlign(tview.AlignLeft)

	cmdView = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true)
	cmdView.SetBorder(true).SetTitle(fmt.Sprintf("[%s] Install Command ", titleColor)).SetTitleAlign(tview.AlignLeft)

	refreshList(0)

	list.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		// This can be used to edit, but we'll use shortcuts instead
	})

	list.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		updateDetails(index)
	})

	flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(list, 0, 1, true).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(cmdView, 0, 1, false).
				AddItem(details, 0, 3, false), 0, 2, false), 0, 1, true).
		AddItem(tview.NewTextView().
			SetTextAlign(tview.AlignCenter).
			SetDynamicColors(true).
			SetText(fmt.Sprintf("[%s]<a> Add  <e> Edit  <d> Delete  <Space> Select  <Enter> Run  <c> Copy  </> Search  <p> Path  <q> Quit", helpColor)), 1, 1, false)

	pages.AddPage("main", flex, true, true)

	tviewApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		frontPage, _ := pages.GetFrontPage()
		if frontPage != "main" {
			return event
		}

		if event.Key() == tcell.KeyEsc {
			// ESC pressed: clear selected items if any
			if len(selectedIndices) > 0 {
				selectedIndices = make(map[string]bool)
				refreshList(list.GetCurrentItem())
				return nil
			}
			// If no selections, reset filter if active
			if lastSearchText != "" {
				resetFilter()
				return nil
			}
			return event
		}

		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case 'q':
				tviewApp.Stop()
				return nil
			case 'p':
				showConfigForm()
				return nil
			case 'a':
				showForm(-1)
				return nil
			case 'e':
				if len(filteredIndices) > 0 {
					showForm(list.GetCurrentItem())
					return nil
				}
			case 'd':
				if len(filteredIndices) > 0 {
					showDeleteConfirm()
					return nil
				}
			case 'c':
				if len(filteredIndices) > 0 {
					copyInstallCmd(list.GetCurrentItem())
					return nil
				}
			case ' ':
				if len(filteredIndices) > 0 {
					toggleSelection(list.GetCurrentItem())
					return nil
				}
			case '/':
				showSearch()
				return nil
			}
		}
		// Enter to run install command
		if event.Key() == tcell.KeyEnter {
			if len(filteredIndices) > 0 {
				showRunConfirm(list.GetCurrentItem())
				return nil
			}
		}
		return event
	})

	if err := tviewApp.SetRoot(pages, true).Run(); err != nil {
		panic(err)
	}
}

func resetFilteredIndices() {
	filteredIndices = nil
	for i := range apps {
		filteredIndices = append(filteredIndices, i)
	}
}

func resetFilter() {
	lastSearchText = ""
	searchBoxVisible = false
	selectedIndices = make(map[string]bool)
	resetFilteredIndices()
	refreshList(0)
}

func showConfigForm() {
	input := tview.NewInputField().
		SetText(app.AppsFilePath).
		SetFieldWidth(40).
		SetFieldBackgroundColor(tcell.ColorReset).SetFieldTextColor(tcell.ColorBlack)
	input.SetBorder(true).
		SetTitle(fmt.Sprintf("[%s] App storage path [json] [white]-----[%s] <Enter> Save <Esc> Cancel ", titleColor, helpColor)).
		SetTitleAlign(tview.AlignLeft)
	input.SetFocusFunc(func() { input.SetBorderColor(borderSelectedColor) })
	input.SetBlurFunc(func() { input.SetBorderColor(borderNormalColor) })

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			pages.RemovePage("config")
			return nil
		}
		if event.Key() == tcell.KeyEnter {
			newPath := input.GetText()
			if newPath != "" {
				app.AppsFilePath = newPath
				app.SaveConfig(app.Config{AppsFile: newPath})
				var err error
				apps, err = app.LoadApps()
				if err != nil {
					// Handle error, maybe show alert, but for now just refresh
				}
				resetFilteredIndices()
				refreshList(0)
				pages.RemovePage("config")
			}
			return nil
		}
		return event
	})

	pages.AddPage("config", modal(input, 60, 3), true, true)
	tviewApp.SetFocus(input)
}

func refreshList(selectedIndex int) {
	list.Clear()
	list.SetTitle(fmt.Sprintf("[%s] Favourite Apps (%d) ", titleColor, len(filteredIndices)))
	for i, idx := range filteredIndices {
		a := apps[idx]

		// Show selection indicator for multi-select
		selectionIndicator := " "
		if selectedIndices[a.Name] {
			selectionIndicator = "[green]✓[black] "
		}

		// Use tview's markup for serial vs name
		list.AddItem(fmt.Sprintf("[gray][%d][black] %s%s ", i+1, selectionIndicator, a.Name), "", 0, nil)
	}

	if len(filteredIndices) > 0 {
		if selectedIndex < 0 {
			selectedIndex = 0
		} else if selectedIndex >= len(filteredIndices) {
			selectedIndex = len(filteredIndices) - 1
		}
		list.SetCurrentItem(selectedIndex)
		updateDetails(selectedIndex)
	} else {
		details.Clear()
		cmdView.Clear()
	}
}

func updateDetails(index int) {
	if index >= 0 && index < len(filteredIndices) {
		details.Clear()
		fmt.Fprintf(details, "[black]%s", apps[filteredIndices[index]].Description)

		cmdView.Clear()
		a := apps[filteredIndices[index]]
		currentCmd := a.Cmd()

		writeOSCmd := func(label, cmd string) {
			if cmd == "" {
				return
			}
			if cmd == currentCmd {
				fmt.Fprintf(cmdView, "[green]%s:[black] %s\n", label, cmd)
			} else {
				fmt.Fprintf(cmdView, "[gray]%s:[black] %s\n", label, cmd)
			}
		}
		writeOSCmd("Mac", a.CmdMac)
		writeOSCmd("Linux", a.CmdLinux)
	}
}

func showSearch() {
	searchBoxVisible = true
	input := tview.NewInputField().
		SetText(lastSearchText).
		SetFieldBackgroundColor(tcell.ColorReset).SetFieldTextColor(tcell.ColorBlack)
	input.SetBorder(true).
		SetTitle(fmt.Sprintf("[%s] Search [white]----- [%s]<Esc> Hide <Enter> Close ", titleColor, helpColor)).
		SetTitleAlign(tview.AlignLeft)
	input.SetFocusFunc(func() { input.SetBorderColor(borderSelectedColor) })
	input.SetBlurFunc(func() { input.SetBorderColor(borderNormalColor) })

	searchInput = input

	input.SetChangedFunc(func(text string) {
		lastSearchText = text
		filteredIndices = nil
		searchLower := strings.ToLower(text)
		for i, a := range apps {
			if text == "" ||
				strings.Contains(strings.ToLower(a.Name), searchLower) ||
				strings.Contains(strings.ToLower(a.Description), searchLower) ||
				strings.Contains(strings.ToLower(a.CmdMac), searchLower) ||
				strings.Contains(strings.ToLower(a.CmdLinux), searchLower) {
				filteredIndices = append(filteredIndices, i)
			}
		}
		refreshList(0)
	})

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			// Hide search box but keep filter active
			searchBoxVisible = false
			pages.RemovePage("search")
			tviewApp.SetFocus(list)
			return nil
		}
		if event.Key() == tcell.KeyEnter || event.Key() == tcell.KeyUp || event.Key() == tcell.KeyDown {
			// Close search but keep filter active
			pages.RemovePage("search")
			tviewApp.SetFocus(list)
			return nil
		}
		return event
	})

	pages.AddPage("search", modal(input, 60, 3), true, true)
	tviewApp.SetFocus(input)
}

func copyInstallCmd(index int) {
	if index < 0 || index >= len(filteredIndices) {
		return
	}
	appIdx := filteredIndices[index]
	cmd := apps[appIdx].Cmd()
	if cmd == "" {
		return
	}
	clipboard.WriteAll(cmd)
	cmdView.Clear()
	fmt.Fprintf(cmdView, "[cyan]%s [green](Copied!)", cmd)
}

func toggleSelection(index int) {
	if index < 0 || index >= len(filteredIndices) {
		return
	}
	appIdx := filteredIndices[index]
	appName := apps[appIdx].Name
	// Toggle selection state
	if selectedIndices[appName] {
		delete(selectedIndices, appName)
	} else {
		selectedIndices[appName] = true
	}
	refreshList(index)
}

func showRunConfirm(index int) {
	// Priority: selected items > highlighted item
	if len(selectedIndices) > 0 {
		// Build list of selected apps with commands
		selectedApps := []struct {
			name string
			cmd  string
		}{}
		for appName := range selectedIndices {
			for _, app := range apps {
				if app.Name == appName && app.Cmd() != "" {
					selectedApps = append(selectedApps, struct {
						name string
						cmd  string
					}{app.Name, app.Cmd()})
					break
				}
			}
		}

		if len(selectedApps) == 0 {
			return
		}

		confirmText := tview.NewTextView().
			SetTextAlign(tview.AlignCenter).
			SetDynamicColors(true)

		var confirmBox *tview.Flex

		if len(selectedApps) == 1 {
			// Single selected item - show its command
			confirmText.SetText(fmt.Sprintf("\nRun this command?\n\n[cyan]%s", selectedApps[0].cmd))
			confirmBox = tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(confirmText, 0, 1, true)
			confirmBox.SetBorder(true).
				SetTitle(fmt.Sprintf("[%s] Confirm [white]----- [%s]<Enter> Run <Esc> Cancel ", titleColor, helpColor)).
				SetTitleAlign(tview.AlignLeft)
		} else {
			// Multiple selected items
			text := fmt.Sprintf("\nRun commands for [green]%d[white] selected apps?\n\n", len(selectedApps))
			for _, app := range selectedApps {
				text += fmt.Sprintf("[yellow]• %s[white]\n", app.name)
			}
			confirmText.SetText(text)
			confirmBox = tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(confirmText, 0, 1, true)
			confirmBox.SetBorder(true).
				SetTitle(fmt.Sprintf("[%s] Confirm [white]----- [%s]<Enter> Run All <Esc> Cancel ", titleColor, helpColor)).
				SetTitleAlign(tview.AlignLeft)
		}

		confirmBox.SetFocusFunc(func() { confirmBox.SetBorderColor(borderSelectedColor) })
		confirmBox.SetBlurFunc(func() { confirmBox.SetBorderColor(borderNormalColor) })

		confirmBox.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEsc {
				pages.RemovePage("runConfirm")
				return nil
			}
			if event.Key() == tcell.KeyEnter {
				pages.RemovePage("runConfirm")
				if len(selectedApps) == 1 {
					// Find and run the selected app's command
					for i, idx := range filteredIndices {
						if apps[idx].Name == selectedApps[0].name {
							runInstallCmd(i)
							break
						}
					}
				} else {
					runAllSelectedCmds()
				}
				return nil
			}
			return event
		})

		height := 10
		if len(selectedApps) > 1 {
			height = 14
		}
		pages.AddPage("runConfirm", modal(confirmBox, 80, height), true, true)
		return
	}

	// No selection, use current highlighted item
	if index < 0 || index >= len(filteredIndices) {
		return
	}
	appIdx := filteredIndices[index]
	cmd := apps[appIdx].Cmd()
	if cmd == "" {
		return
	}

	confirmText := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText(fmt.Sprintf("\nRun this command?\n\n[cyan]%s", cmd))

	confirmBox := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(confirmText, 0, 1, true)
	confirmBox.SetBorder(true).
		SetTitle(fmt.Sprintf("[%s] Confirm [white]----- [%s]<Enter> Run <Esc> Cancel ", titleColor, helpColor)).
		SetTitleAlign(tview.AlignLeft)
	confirmBox.SetFocusFunc(func() { confirmBox.SetBorderColor(borderSelectedColor) })
	confirmBox.SetBlurFunc(func() { confirmBox.SetBorderColor(borderNormalColor) })

	confirmBox.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			pages.RemovePage("runConfirm")
			return nil
		}
		if event.Key() == tcell.KeyEnter {
			pages.RemovePage("runConfirm")
			runInstallCmd(index)
			return nil
		}
		return event
	})

	pages.AddPage("runConfirm", modal(confirmBox, 80, 10), true, true)
}

func runAllSelectedCmds() {
	// Collect all commands from selected items
	type appCmd struct {
		name string
		cmd  string
	}
	var commands []appCmd

	for appName := range selectedIndices {
		for _, app := range apps {
			if app.Name == appName && app.Cmd() != "" {
				commands = append(commands, appCmd{app.Name, app.Cmd()})
				break
			}
		}
	}

	if len(commands) == 0 {
		return
	}

	// Create output view
	outputView := tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true)
	outputView.SetBorder(true).
		SetTitle(fmt.Sprintf("[%s] Running Commands [white]----- [%s]<Esc> Close ", titleColor, helpColor)).
		SetTitleAlign(tview.AlignLeft)
	outputView.ScrollToEnd()

	outputView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			pages.RemovePage("runCmd")
			return nil
		}
		return event
	})

	modalView := modal(outputView, 80, 24)
	pages.AddPage("runCmd", modalView, true, true)

	// Run all commands sequentially
	go func() {
		for i, ac := range commands {
			tviewApp.QueueUpdateDraw(func() {
				fmt.Fprintf(outputView, "\n[cyan][%d/%d] %s[white]\n", i+1, len(commands), ac.name)
				fmt.Fprintf(outputView, "[gray]%s[white]\n", ac.cmd)
			})

			cmdExec := exec.Command("sh", "-c", ac.cmd)
			output, err := cmdExec.CombinedOutput()
			outputStr := string(output)

			tviewApp.QueueUpdateDraw(func() {
				alreadyInstalledPatterns := []string{
					"already installed",
					"is already installed",
					"up-to-date",
					"up to date",
					"no action needed",
					"latest version",
				}

				isAlreadyInstalled := false
				outputLower := strings.ToLower(outputStr)
				for _, pattern := range alreadyInstalledPatterns {
					if strings.Contains(outputLower, pattern) {
						isAlreadyInstalled = true
						break
					}
				}

				if isAlreadyInstalled {
					fmt.Fprintf(outputView, "[yellow]  Already installed![white]\n")
				} else if err != nil {
					fmt.Fprintf(outputView, "[red]  Error: %v[white]\n", err)
				} else {
					fmt.Fprintf(outputView, "[green]  Success![white]\n")
				}
			})
		}

		tviewApp.QueueUpdateDraw(func() {
			fmt.Fprintf(outputView, "\n[green]All commands completed![white]\n")
		})
	}()

	tviewApp.SetFocus(outputView)
}

func runInstallCmd(index int) {
	if index < 0 || index >= len(filteredIndices) {
		return
	}
	appIdx := filteredIndices[index]
	cmd := apps[appIdx].Cmd()
	if cmd == "" {
		return
	}

	// Create output view for command execution
	outputView := tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true)
	outputView.SetBorder(true).
		SetTitle(fmt.Sprintf("[%s] Running Command [white]----- [%s]<Esc> Close ", titleColor, helpColor)).
		SetTitleAlign(tview.AlignLeft)

	outputView.ScrollToEnd()

	// Set up input capture for the output view
	outputView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			pages.RemovePage("runCmd")
			return nil
		}
		return event
	})

	// Create modal
	modalView := modal(outputView, 80, 20)
	pages.AddPage("runCmd", modalView, true, true)

	// Run command in goroutine
	go func() {
		tviewApp.QueueUpdateDraw(func() {
			fmt.Fprintf(outputView, "[yellow]Executing...[white]\n")
		})

		cmdExec := exec.Command("sh", "-c", cmd)
		output, err := cmdExec.CombinedOutput()
		outputStr := string(output)

		tviewApp.QueueUpdateDraw(func() {
			// Check if already installed (common package manager messages)
			alreadyInstalledPatterns := []string{
				"already installed",
				"is already installed",
				"up-to-date",
				"up to date",
				"no action needed",
				"latest version",
			}

			isAlreadyInstalled := false
			outputLower := strings.ToLower(outputStr)
			for _, pattern := range alreadyInstalledPatterns {
				if strings.Contains(outputLower, pattern) {
					isAlreadyInstalled = true
					break
				}
			}

			if isAlreadyInstalled {
				fmt.Fprintf(outputView, "\n[yellow]Already installed![white]\n")
			} else if err != nil {
				fmt.Fprintf(outputView, "\n[red]Error:[white] %v\n", err)
			} else {
				fmt.Fprintf(outputView, "\n[green]Success![white]\n")
			}
			if len(outputStr) > 0 {
				fmt.Fprintf(outputView, "\n[cyan]Output:[white]\n%s", outputStr)
			}
		})
	}()

	tviewApp.SetFocus(outputView)
}

func showForm(index int) {
	name := ""
	description := ""
	cmdMac := ""
	cmdLinux := ""

	if index >= 0 && index < len(filteredIndices) {
		name = apps[filteredIndices[index]].Name
		description = apps[filteredIndices[index]].Description
		cmdMac = apps[filteredIndices[index]].CmdMac
		cmdLinux = apps[filteredIndices[index]].CmdLinux
	}

	nameInput := tview.NewInputField().
		SetText(name).
		SetFieldBackgroundColor(tcell.ColorReset).SetFieldTextColor(tcell.ColorBlack)
	nameInput.SetBorder(true).SetTitle(fmt.Sprintf("[%s] Name ", titleColor)).SetTitleAlign(tview.AlignLeft)
	nameInput.SetFocusFunc(func() { nameInput.SetBorderColor(borderSelectedColor) })
	nameInput.SetBlurFunc(func() { nameInput.SetBorderColor(borderNormalColor) })

	descInput := tview.NewTextArea(). // SetPlaceholderStyle(tcell.StyleDefault.Foreground(tcell.ColorGray)).
						SetText(description, false).SetTextStyle(tcell.StyleDefault.Foreground(tcell.ColorBlack))
	descInput.SetBorder(true).SetTitle(fmt.Sprintf("[%s] Description [white]----- [%s]<Ctrl+s> Save <Esc> Cancel ", titleColor, helpColor)).SetTitleAlign(tview.AlignLeft)
	descInput.SetFocusFunc(func() { descInput.SetBorderColor(borderSelectedColor) })
	descInput.SetBlurFunc(func() { descInput.SetBorderColor(borderNormalColor) })

	cmdMacInput := tview.NewInputField().
		SetText(cmdMac).
		SetFieldBackgroundColor(tcell.ColorReset).SetFieldTextColor(tcell.ColorBlack)

	cmdMacInput.SetBorder(true).SetTitle(fmt.Sprintf("[%s] Install Cmd (Mac) ", titleColor)).SetTitleAlign(tview.AlignLeft)
	cmdMacInput.SetFocusFunc(func() { cmdMacInput.SetBorderColor(borderSelectedColor) })
	cmdMacInput.SetBlurFunc(func() { cmdMacInput.SetBorderColor(borderNormalColor) })

	cmdLinuxInput := tview.NewInputField().
		SetText(cmdLinux).
		SetFieldBackgroundColor(tcell.ColorReset).SetFieldTextColor(tcell.ColorBlack)

	cmdLinuxInput.SetBorder(true).SetTitle(fmt.Sprintf("[%s] Install Cmd (Linux) ", titleColor)).SetTitleAlign(tview.AlignLeft)
	cmdLinuxInput.SetFocusFunc(func() { cmdLinuxInput.SetBorderColor(borderSelectedColor) })
	cmdLinuxInput.SetBlurFunc(func() { cmdLinuxInput.SetBorderColor(borderNormalColor) })

	errorText := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	// Layout like lazygit commit message
	formFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nameInput, 3, 1, true).
		AddItem(descInput, 0, 1, false).
		AddItem(cmdMacInput, 3, 1, false).
		AddItem(cmdLinuxInput, 3, 1, false).
		AddItem(errorText, 2, 1, false)

	// Input capture for name input to handle Enter
	nameInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			if nameInput.GetText() == "" {
				errorText.SetText("[red]Name cannot be empty")
				return nil
			}
			errorText.SetText("")
			tviewApp.SetFocus(descInput)
			return nil
		}
		return event
	})

	// Input capture for the form
	formFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			pages.RemovePage("form")
			return nil
		}
		// Ctrl+S to save
		if event.Key() == tcell.KeyCtrlS {
			newName := nameInput.GetText()
			newDesc := descInput.GetText()
			newCmdMac := cmdMacInput.GetText()
			newCmdLinux := cmdLinuxInput.GetText()
			if newName == "" {
				errorText.SetText("[red]Name cannot be empty")
				return nil
			}

			// Check for duplicate name
			for i, a := range apps {
				if index >= 0 && i == filteredIndices[index] {
					continue // skip current app when editing
				}
				if strings.EqualFold(a.Name, newName) {
					errorText.SetText("[red]Name already exists")
					return nil
				}
			}

			newApp := app.App{Name: newName, Description: newDesc, CmdMac: newCmdMac, CmdLinux: newCmdLinux}
			if index >= 0 {
				apps[filteredIndices[index]] = newApp
			} else {
				apps = append(apps, newApp)
				index = len(apps) - 1
			}
			app.SaveApps(apps)
			resetFilteredIndices()
			refreshList(index)
			pages.RemovePage("form")
			return nil
		}
		// Tab to switch between fields (legacy support)
		if event.Key() == tcell.KeyTab {
			if nameInput.HasFocus() {
				tviewApp.SetFocus(descInput)
			} else if descInput.HasFocus() {
				tviewApp.SetFocus(cmdMacInput)
			} else if cmdMacInput.HasFocus() {
				tviewApp.SetFocus(cmdLinuxInput)
			} else {
				tviewApp.SetFocus(nameInput)
			}
			return nil
		}
		return event
	})

	pages.AddPage("form", modal(formFlex, 60, 23), true, true)
	tviewApp.SetFocus(nameInput)
}

func showDeleteConfirm() {
	// Check if there are selected items (multi-select)
	if len(selectedIndices) > 0 {
		// Build list of selected app names for confirmation
		selectedNames := []string{}
		for appName := range selectedIndices {
			selectedNames = append(selectedNames, appName)
		}

		confirmText := tview.NewTextView().
			SetTextAlign(tview.AlignCenter).
			SetDynamicColors(true)

		if len(selectedNames) == 1 {
			confirmText.SetText(fmt.Sprintf("\nAre you sure you want to delete\n\n[yellow]%s[white]?", selectedNames[0]))
		} else {
			namesList := ""
			for _, name := range selectedNames {
				namesList += fmt.Sprintf("\n[yellow]• %s[white]", name)
			}
			confirmText.SetText(fmt.Sprintf("\nAre you sure you want to delete [red]%d[white] apps?%s", len(selectedNames), namesList))
		}

		confirmBox := tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(confirmText, 0, 1, true)
		confirmBox.SetBorder(true).
			SetTitle(fmt.Sprintf("[%s] Delete [white]----- [%s]<Enter> Confirm <Esc> Cancel ", titleColor, helpColor)).
			SetTitleAlign(tview.AlignLeft)
		confirmBox.SetFocusFunc(func() { confirmBox.SetBorderColor(borderSelectedColor) })
		confirmBox.SetBlurFunc(func() { confirmBox.SetBorderColor(borderNormalColor) })

		confirmBox.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEsc {
				pages.RemovePage("delete")
				return nil
			}
			if event.Key() == tcell.KeyEnter {
				// Delete selected items by name
				for appName := range selectedIndices {
					for i, app := range apps {
						if app.Name == appName {
							apps = append(apps[:i], apps[i+1:]...)
							break
						}
					}
				}

				app.SaveApps(apps)
				selectedIndices = make(map[string]bool) // Clear selection
				resetFilteredIndices()
				refreshList(0)
				pages.RemovePage("delete")
			}
			return event
		})

		pages.AddPage("delete", modal(confirmBox, 60, 12), true, true)
		return
	}

	// No selected items, delete current item (existing behavior)
	index := list.GetCurrentItem()
	if index < 0 || index >= len(filteredIndices) {
		return
	}

	appIdx := filteredIndices[index]
	confirmText := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText(fmt.Sprintf("\nAre you sure you want to delete\n\n[yellow]%s[white]?", apps[appIdx].Name))

	confirmBox := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(confirmText, 0, 1, true)
	confirmBox.SetBorder(true).
		SetTitle(fmt.Sprintf("[%s] Delete [white]----- [%s]<Enter> Confirm <Esc> Cancel ", titleColor, helpColor)).
		SetTitleAlign(tview.AlignLeft)
	confirmBox.SetFocusFunc(func() { confirmBox.SetBorderColor(borderSelectedColor) })
	confirmBox.SetBlurFunc(func() { confirmBox.SetBorderColor(borderNormalColor) })

	confirmBox.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			pages.RemovePage("delete")
			return nil
		}
		if event.Key() == tcell.KeyEnter {
			apps = append(apps[:appIdx], apps[appIdx+1:]...)
			app.SaveApps(apps)
			resetFilteredIndices()
			refreshList(index - 1)
			pages.RemovePage("delete")
		}
		return event
	})

	pages.AddPage("delete", modal(confirmBox, 60, 8), true, true)
}

func modal(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)
}
