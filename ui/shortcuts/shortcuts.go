package shortcuts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

var (
	ShortcutReload      = desktop.CustomShortcut{KeyName: fyne.KeyR, Modifier: fyne.KeyModifierShortcutDefault}
	ShortcutSearch      = desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: fyne.KeyModifierShortcutDefault}
	ShortcutQuickSearch = desktop.CustomShortcut{KeyName: fyne.KeyG, Modifier: fyne.KeyModifierShortcutDefault}
	ShortcutCloseWindow = desktop.CustomShortcut{KeyName: fyne.KeyW, Modifier: fyne.KeyModifierShortcutDefault}

	ShortcutNavOne   = desktop.CustomShortcut{KeyName: fyne.Key1, Modifier: fyne.KeyModifierShortcutDefault}
	ShortcutNavTwo   = desktop.CustomShortcut{KeyName: fyne.Key2, Modifier: fyne.KeyModifierShortcutDefault}
	ShortcutNavThree = desktop.CustomShortcut{KeyName: fyne.Key3, Modifier: fyne.KeyModifierShortcutDefault}
	ShortcutNavFour  = desktop.CustomShortcut{KeyName: fyne.Key4, Modifier: fyne.KeyModifierShortcutDefault}
	ShortcutNavFive  = desktop.CustomShortcut{KeyName: fyne.Key5, Modifier: fyne.KeyModifierShortcutDefault}
	ShortcutNavSix   = desktop.CustomShortcut{KeyName: fyne.Key6, Modifier: fyne.KeyModifierShortcutDefault}
	ShortcutNavSeven = desktop.CustomShortcut{KeyName: fyne.Key7, Modifier: fyne.KeyModifierShortcutDefault}
	ShortcutNavEight = desktop.CustomShortcut{KeyName: fyne.Key8, Modifier: fyne.KeyModifierShortcutDefault}

	NavShortcuts = []desktop.CustomShortcut{ShortcutNavOne, ShortcutNavTwo, ShortcutNavThree,
		ShortcutNavFour, ShortcutNavFive, ShortcutNavSix, ShortcutNavSeven, ShortcutNavEight}
)
