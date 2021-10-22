package gallium

/*
#include <stdlib.h>
#include "gallium/gallium.h"
#include "gallium/menu.h"

// It does not seem that we can import "_cgo_export.h" from here
extern void cgo_onMenuClicked(void*);

// This is a wrapper around NSMenu_AddMenuItem that adds the function pointer
// argument, since this does not seem to be possible from Go directly.
static inline gallium_nsmenuitem_t* helper_NSMenu_AddMenuItem(
	gallium_nsmenu_t* menu,
	const char* title,
	const char* shortcutKey,
	gallium_modifier_t shortcutModifier,
	void *callbackArg) {

	return NSMenu_AddMenuItem(
		menu,
		title,
		shortcutKey,
		shortcutModifier,
		&cgo_onMenuClicked,
		callbackArg);
}

*/
import "C"
import (
	"errors"
	"fmt"
	"log"
	"strings"
)

type MenuEntry interface {
	menu()
}

type MenuItem struct {
	Title    string
	Shortcut string
	OnClick  func()
}

func (MenuItem) menu() {}

type Menu struct {
	Title   string
	Entries []MenuEntry
}

func (Menu) menu() {}

var menuMgr *menuManager

type menuManager struct {
	items map[int]MenuItem
}

func newMenuManager() *menuManager {
	return &menuManager{make(map[int]MenuItem)}
}

func (m *menuManager) add(menu MenuEntry, parent *C.gallium_nsmenu_t) {
	switch menu := menu.(type) {
	case Menu:
		item := C.NSMenu_AddMenuItem(parent, C.CString(menu.Title), nil, 0, nil, nil)
		submenu := C.NSMenu_New(C.CString(menu.Title))
		C.NSMenuItem_SetSubmenu(item, submenu)
		for _, entry := range menu.Entries {
			m.add(entry, submenu)
		}
	case MenuItem:
		id := len(m.items)
		m.items[id] = menu

		callbackArg := C.malloc(C.sizeof_int)
		*(*C.int)(callbackArg) = C.int(id)

		key, modifiers, _ := parseShortcut(menu.Shortcut)

		C.helper_NSMenu_AddMenuItem(
			parent,
			C.CString(menu.Title),
			C.CString(key),
			C.gallium_modifier_t(modifiers),
			callbackArg)
	default:
		log.Printf("unexpected menu entry: %T", menu)
	}
}

func parseShortcut(s string) (key string, modifiers int, err error) {
	parts := strings.Split(s, "+")
	if len(parts) == 0 {
		return "", 0, fmt.Errorf("empty shortcut")
	}
	key = parts[len(parts)-1]
	if len(key) == 0 {
		return "", 0, fmt.Errorf("empty key")
	}
	for _, part := range parts[:len(parts)-1] {
		switch strings.ToLower(part) {
		case "cmd":
			modifiers |= int(C.GalliumCmdModifier)
		case "ctrl":
			modifiers |= int(C.GalliumCmdModifier)
		case "cmdctrl":
			modifiers |= int(C.GalliumCmdOrCtrlModifier)
		case "alt":
			modifiers |= int(C.GalliumAltOrOptionModifier)
		case "option":
			modifiers |= int(C.GalliumAltOrOptionModifier)
		case "fn":
			modifiers |= int(C.GalliumFunctionModifier)
		case "shift":
			modifiers |= int(C.GalliumShiftModifier)
		default:
			return "", 0, fmt.Errorf("unknown modifier: %s", part)
		}
	}
	return
}

func (app *App) SetMenu(menus []Menu) {
	if menuMgr == nil {
		menuMgr = newMenuManager()
	}
	root := C.NSMenu_New(C.CString("<root>"))
	for _, m := range menus {
		menuMgr.add(m, root)
	}
	C.NSApplication_SetMainMenu(root)
}

func (app *App) AddStatusItem(width int, title string, highlight bool, entries ...MenuEntry) {
	if menuMgr == nil {
		menuMgr = newMenuManager()
	}

	root := C.NSMenu_New(C.CString("<statusbar>"))
	for _, m := range entries {
		menuMgr.add(m, root)
	}
	C.NSStatusBar_AddItem(C.int(width), C.CString(title), C.bool(highlight), root)
}

// Image holds a handle to a platform-specific image structure. On OSX it is NSImage.
type Image struct {
	c *C.gallium_nsimage_t
}

var (
	ErrImageDecodeFailed = errors.New("image could not be decoded")
)

// ImageFromPNG creates an image from a buffer containing a PNG-encoded image.
func ImageFromPNG(buf []byte) (*Image, error) {
	cbuf := C.CBytes(buf)
	defer C.free(cbuf)
	cimg := C.NSImage_NewFromPNG(cbuf, C.int(len(buf)))
	if cimg == nil {
		return nil, ErrImageDecodeFailed
	}
	return &Image{cimg}, nil
}

// ImageToPNG writes an image to the given file
func ImageToPNG(image *Image, path string) {
	C.NSImage_WriteToFile(image.c, C.CString(path))
}

type Notification struct {
	Title             string
	Subtitle          string
	InformativeText   string
	Image             *Image
	Identifier        string
	ActionButtonTitle string
	OtherButtonTitle  string
}

func (app *App) Post(n Notification) {
	var cimg *C.gallium_nsimage_t
	if n.Image != nil {
		cimg = n.Image.c
	}
	cn := C.NSUserNotification_New(
		C.CString(n.Title),
		C.CString(n.Subtitle),
		C.CString(n.InformativeText),
		cimg,
		C.CString(n.Identifier),
		len(n.ActionButtonTitle) > 0,
		len(n.OtherButtonTitle) > 0,
		C.CString(n.ActionButtonTitle),
		C.CString(n.OtherButtonTitle))

	C.NSUserNotificationCenter_DeliverNotification(cn)
}

func RunApplication() {
	log.Println("in RunApplication")
	C.NSApplication_Run()
}
