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
	list.SetBorder(true).SetTitle(" Favourite Apps ")

	details = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)
	details.SetBorder(true).SetTitle(" Description ")

	refreshList()

	list.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		// This can be used to edit, but we'll use shortcuts instead
	})

	list.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		updateDetails(index)
	})

	flex = tview.NewFlex().
		AddItem(list, 0, 1, true).
		AddItem(details, 0, 2, false)

	pages.AddPage("main", flex, true, true)

	tviewApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case 'q':
				tviewApp.Stop()
			case 'a':
				showForm(-1)
			case 'e':
				if len(apps) > 0 {
					showForm(list.GetCurrentItem())
				}
			case 'd':
				if len(apps) > 0 {
					showDeleteConfirm()
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
	for i, a := range apps {
		list.AddItem(a.Name, "", 0, nil)
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
	form := tview.NewForm()
	name := ""
	description := ""

	if index >= 0 && index < len(apps) {
		name = apps[index].Name
		description = apps[index].Description
		form.SetTitle(" Edit App ")
	} else {
		index = -1 // Ensure it's treated as "Add" if out of bounds
		form.SetTitle(" Add App ")
	}

	form.AddInputField("Name", name, 20, nil, func(text string) {
		name = text
	})
	form.AddTextArea("Description", description, 40, 5, 0, func(text string) {
		description = text
	})

	form.AddButton("Save", func() {
		newApp := app.App{Name: name, Description: description}
		if index >= 0 {
			apps[index] = newApp
		} else {
			apps = append(apps, newApp)
		}
		app.SaveApps(apps)
		refreshList()
		pages.RemovePage("form")
	})
	form.AddButton("Cancel", func() {
		pages.RemovePage("form")
	})

	form.SetBorder(true)
	pages.AddPage("form", modal(form, 60, 15), true, true)
}

func showDeleteConfirm() {
	index := list.GetCurrentItem()
	if index < 0 || index >= len(apps) {
		return
	}

	modal := tview.NewModal().
		SetText(fmt.Sprintf("Are you sure you want to delete '%s'?", apps[index].Name)).
		AddButtons([]string{"Delete", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Delete" {
				apps = append(apps[:index], apps[index+1:]...)
				app.SaveApps(apps)
				refreshList()
			}
			pages.RemovePage("delete")
		})

	pages.AddPage("delete", modal, true, true)
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
