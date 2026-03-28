package main

import (
	"favapss/app"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	tviewApp        *tview.Application
	list            *tview.List
	details         *tview.TextView
	flex            *tview.Flex
	pages           *tview.Pages
	apps            []app.App
	filteredIndices []int

	titleColor          = "green"
	helpColor           = "yellow"
	borderNormalColor   = tcell.ColorWhite
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

	tviewApp = tview.NewApplication()
	pages = tview.NewPages()

	tview.Styles.BorderColor = borderNormalColor

	list = tview.NewList().ShowSecondaryText(false).
		SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorGreen).Foreground(tcell.ColorBlack))
	list.SetBorder(true).SetTitle(fmt.Sprintf("[%s] Favourite Apps ", titleColor)).SetTitleAlign(tview.AlignLeft)
	list.SetFocusFunc(func() { list.SetBorderColor(borderSelectedColor) })
	list.SetBlurFunc(func() { list.SetBorderColor(borderNormalColor) })

	details = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)
	details.SetBorder(true).SetTitle(fmt.Sprintf("[%s] Description ", titleColor)).SetTitleAlign(tview.AlignLeft)

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
			AddItem(details, 0, 2, false), 0, 1, true).
		AddItem(tview.NewTextView().
			SetTextAlign(tview.AlignCenter).
			SetDynamicColors(true).
			SetText(fmt.Sprintf("[%s]<a> Add  <e> Edit  <d> Delete  <p> data path  </> Search  <q> Quit", helpColor)), 1, 1, false)

	pages.AddPage("main", flex, true, true)

	tviewApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		frontPage, _ := pages.GetFrontPage()
		if frontPage != "main" {
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
			case '/':
				showSearch()
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

func showConfigForm() {
	input := tview.NewInputField().
		SetText(app.AppsFilePath).
		SetFieldWidth(40).
		SetFieldBackgroundColor(tcell.ColorReset)
	input.SetBorder(true).
		SetTitle(fmt.Sprintf("[%s] App storage path [json]====== [%s] <Enter> Save <Esc> Cancel ", titleColor, helpColor)).
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
		// Use tview's markup for serial vs name
		list.AddItem(fmt.Sprintf("[gray][%d] [white] %s  ", i+1, a.Name), "", 0, nil)
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
	}
}

func updateDetails(index int) {
	if index >= 0 && index < len(filteredIndices) {
		details.Clear()
		fmt.Fprintf(details, "%s", apps[filteredIndices[index]].Description)
	}
}

func showSearch() {
	input := tview.NewInputField().
		SetFieldBackgroundColor(tcell.ColorReset)
	input.SetBorder(true).
		SetTitle(fmt.Sprintf("[%s] Search [%s] <Esc> Close ", titleColor, helpColor)).
		SetTitleAlign(tview.AlignLeft)
	input.SetFocusFunc(func() { input.SetBorderColor(borderSelectedColor) })
	input.SetBlurFunc(func() { input.SetBorderColor(borderNormalColor) })

	input.SetChangedFunc(func(text string) {
		filteredIndices = nil
		for i, a := range apps {
			if text == "" ||
				strings.Contains(strings.ToLower(a.Name), strings.ToLower(text)) ||
				strings.Contains(strings.ToLower(a.Description), strings.ToLower(text)) {
				filteredIndices = append(filteredIndices, i)
			}
		}
		refreshList(0)
	})

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc || event.Key() == tcell.KeyEnter {
			pages.RemovePage("search")
			return nil
		}
		return event
	})

	pages.AddPage("search", modal(input, 60, 3), true, true)
	tviewApp.SetFocus(input)
}

func showForm(index int) {
	name := ""
	description := ""

	if index >= 0 && index < len(filteredIndices) {
		name = apps[filteredIndices[index]].Name
		description = apps[filteredIndices[index]].Description
	}

	nameInput := tview.NewInputField().
		SetText(name).
		SetFieldBackgroundColor(tcell.ColorReset)
	nameInput.SetBorder(true).SetTitle(fmt.Sprintf("[%s] Name ", titleColor)).SetTitleAlign(tview.AlignLeft)
	nameInput.SetFocusFunc(func() { nameInput.SetBorderColor(borderSelectedColor) })
	nameInput.SetBlurFunc(func() { nameInput.SetBorderColor(borderNormalColor) })

	descInput := tview.NewTextArea().
		SetPlaceholder("Enter description...").
		SetPlaceholderStyle(tcell.StyleDefault.Foreground(tcell.ColorGray)).
		SetText(description, false)
	descInput.SetBorder(true).SetTitle(fmt.Sprintf("[%s] Description [%s]<Ctrl+s> Save <Esc> Cancel ", titleColor, helpColor)).SetTitleAlign(tview.AlignLeft)
	descInput.SetFocusFunc(func() { descInput.SetBorderColor(borderSelectedColor) })
	descInput.SetBlurFunc(func() { descInput.SetBorderColor(borderNormalColor) })

	// Layout like lazygit commit message
	formFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nameInput, 3, 1, true).
		AddItem(descInput, 0, 1, false)
	// Input capture for name input to handle Enter
	nameInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
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
			if newName == "" {
				return nil
			}

			newApp := app.App{Name: newName, Description: newDesc}
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
			} else {
				tviewApp.SetFocus(nameInput)
			}
			return nil
		}
		return event
	})

	pages.AddPage("form", modal(formFlex, 60, 20), true, true)
	tviewApp.SetFocus(nameInput)
}

func showDeleteConfirm() {
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
		SetTitle(fmt.Sprintf("[%s] Delete [%s] <Enter> Confirm <Esc> Cancel ", titleColor, helpColor)).
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
			return nil
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
