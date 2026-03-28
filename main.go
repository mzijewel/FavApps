package main

import (
	"favapss/app"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	tviewApp *tview.Application
	list     *tview.List
	details  *tview.TextView
	flex     *tview.Flex
	pages    *tview.Pages
	apps     []app.App

	titleColor = "green"
	helpColor  = "yellow"
)

func main() {
	var err error
	apps, err = app.LoadApps()
	if err != nil {
		panic(err)
	}

	tviewApp = tview.NewApplication()
	pages = tview.NewPages()

	list = tview.NewList().ShowSecondaryText(false)
	list.SetBorder(true).SetTitle(fmt.Sprintf("[%s] Favourite Apps ", titleColor)).SetTitleAlign(tview.AlignLeft)

	details = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)
	details.SetBorder(true).SetTitle(fmt.Sprintf("[%s] Description ", titleColor)).SetTitleAlign(tview.AlignLeft)

	refreshList()

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
			SetText(fmt.Sprintf("[%s](a) Add  (e) Edit  (d) Delete  (q) Quit", helpColor)), 1, 1, false)

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
			case 'a':
				showForm(-1)
				return nil
			case 'e':
				if len(apps) > 0 {
					showForm(list.GetCurrentItem())
					return nil
				}
			case 'd':
				if len(apps) > 0 {
					showDeleteConfirm()
					return nil
				}
			}
		}
		return event
	})

	if err := tviewApp.SetRoot(pages, true).Run(); err != nil {
		panic(err)
	}
}

func refreshList() {
	list.Clear()
	list.SetTitle(fmt.Sprintf("[%s] Favourite Apps (%d) ", titleColor, len(apps)))
	for i, a := range apps {
		// Use tview's markup for serial vs name
		list.AddItem(fmt.Sprintf("[black][%d] [white] %s", i+1, a.Name), "", 0, nil)
		if i == 0 {
			updateDetails(0)
		}
	}
	if len(apps) == 0 {
		details.Clear()
	}
}

func updateDetails(index int) {
	if index >= 0 && index < len(apps) {
		details.Clear()
		fmt.Fprintf(details, "%s", apps[index].Description)
	}
}

func showForm(index int) {
	name := ""
	description := ""

	if index >= 0 && index < len(apps) {
		name = apps[index].Name
		description = apps[index].Description
	}

	nameInput := tview.NewInputField().
		SetText(name).
		SetFieldBackgroundColor(tcell.ColorReset)
	nameInput.SetBorder(true).SetTitle(fmt.Sprintf("[%s] Name ", titleColor)).SetTitleAlign(tview.AlignLeft)

	descInput := tview.NewTextArea().
		SetPlaceholder("Enter description...").
		SetText(description, false)
	descInput.SetBorder(true).SetTitle(fmt.Sprintf("[%s] Description [%s]<Ctrl+s> Save <Esc> Cancel ", titleColor, helpColor)).SetTitleAlign(tview.AlignLeft)

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
				apps[index] = newApp
			} else {
				apps = append(apps, newApp)
			}
			app.SaveApps(apps)
			refreshList()
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
	if index < 0 || index >= len(apps) {
		return
	}

	confirmText := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText(fmt.Sprintf("\nAre you sure you want to delete\n\n[yellow]%s[white]?", apps[index].Name))

	confirmBox := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(confirmText, 0, 1, true)
	confirmBox.SetBorder(true).
		SetTitle(fmt.Sprintf("[%s] Delete [%s] <Enter> Confirm <Esc> Cancel ", titleColor, helpColor)).
		SetTitleAlign(tview.AlignLeft)

	confirmBox.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			pages.RemovePage("delete")
			return nil
		}
		if event.Key() == tcell.KeyEnter {
			apps = append(apps[:index], apps[index+1:]...)
			app.SaveApps(apps)
			refreshList()
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
