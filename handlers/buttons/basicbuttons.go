package buttons

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	betterguid "github.com/kjk/betterguid"
)

// Button is a struct that handles button related information
type Button struct {
	URL        bool
	Text       string // text to display
	Data       string // data to get with callback
	uniqueData string // UUID pass to telegram (to ensure that we will not missread another button of this kind)
}

// IgnoreButtonData is a constant for inactive buttons
const IgnoreButtonData = "@ignore"

// NewButton constructs a Button struct
func NewButton(text string, data string) Button {
	return Button{
		URL:        false,
		Text:       text,
		Data:       data,
		uniqueData: betterguid.New(),
	}
}

// NewExternalURLButton creates a Button struct for ButtonURL
func NewExternalURLButton(text string, url string) Button {
	return Button{
		URL:        true,
		Text:       text,
		Data:       url,
		uniqueData: url,
	}
}

// NewIgnoreButton creates a button with standard callback data IgnoreButtonData
func NewIgnoreButton(text string) Button {
	return NewButton(text, IgnoreButtonData)
}

// NewEmptyIgnoreButton creates a button with empty text and with data that should be ignored
func NewEmptyIgnoreButton() Button {
	return NewIgnoreButton(" ")
}

// NewStaticButton creates a button with uniqueData == Data to enable
func NewStaticButton(text string, data string) Button {
	return Button{
		URL:        false,
		Text:       text,
		Data:       data,
		uniqueData: data,
	}
}

// ButtonRow represents a row of buttons
type ButtonRow []Button

// ButtonSet represents list of ButtonRows
type ButtonSet struct {
	rows        []ButtonRow
	dataMapping map[string]string
}

// NewButtonRow creates a ButtonRow from a slice of buttons
func NewButtonRow(buttons ...Button) ButtonRow {
	return buttons
}

// NewSingleRowButtonSet creates a single-row ButtonSet from a slice of buttons
func NewSingleRowButtonSet(buttons ...Button) ButtonSet {
	return ButtonSet{
		rows: []ButtonRow{NewButtonRow(buttons...)},
	}
}

// NewSingleColoumnButtonSet creates a single-coloumn ButtonSet from a slice of buttons
func NewSingleColoumnButtonSet(buttons ...Button) ButtonSet {
	var rows []ButtonRow
	for _, button := range buttons {
		rows = append(rows, NewButtonRow(button))
	}
	return ButtonSet{
		rows: rows,
	}
}

// NewButtonSet constructs ButtonSet from a slice of ButtonRow
func NewButtonSet(rows ...ButtonRow) ButtonSet {
	return ButtonSet{
		rows: rows[:],
	}
}

// EmptyButtonSet returns an empty button set
func EmptyButtonSet() ButtonSet {
	return ButtonSet{}
}

// Join joins slice of ButtonRow with the ButtonSet and returns joined set
func (bs ButtonSet) Join(rows ...ButtonRow) ButtonSet {
	return ButtonSet{
		rows: append(bs.rows, rows...),
	}
}

// JoinSet joins slice of ButtonSet with the current ButtonSet and returns joined set
func (bs ButtonSet) JoinSet(sets ...ButtonSet) ButtonSet {
	rows := bs.rows
	for _, s := range sets {
		rows = append(rows, s.rows...)
	}
	return ButtonSet{
		rows: rows,
	}
}

// emptyReplyKeyboardMarkup returns an empty keyboardMarkup
func emptyReplyKeyboardMarkup() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: make([][]tgbotapi.InlineKeyboardButton, 0)}
}

// GetInlineKeyboard builds tgbotapi.InlineKeyboardMarkup from ButtonSet
func (bs ButtonSet) GetInlineKeyboard() tgbotapi.InlineKeyboardMarkup {
	keyboard := emptyReplyKeyboardMarkup()
	for _, buttonRow := range bs.rows {
		var row []tgbotapi.InlineKeyboardButton
		for _, button := range buttonRow {
			if button.URL {
				row = append(row, tgbotapi.NewInlineKeyboardButtonURL(button.Text, button.Data))
			} else {
				row = append(row, tgbotapi.NewInlineKeyboardButtonData(button.Text, button.uniqueData))
			}
		}
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)
	}
	return keyboard
}

// FindButtonData returns data of a button pressed by a user, or error such button is not present in the set
func (bs ButtonSet) FindButtonData(callbackData string) (string, error) {
	if len(bs.rows) > 0 {
		if len(bs.dataMapping) == 0 {
			bs.dataMapping = make(map[string]string)
			for _, buttonRow := range bs.rows {
				for _, button := range buttonRow {
					bs.dataMapping[button.uniqueData] = button.Data
				}
			}
		}
		if data, ok := bs.dataMapping[callbackData]; ok {
			return data, nil
		}
	}
	return "", fmt.Errorf("callback data %s cannot be found in the button set", callbackData)
}

// IsEmpty indicates that ButtonSet is empty
func (bs ButtonSet) IsEmpty() bool {
	return len(bs.rows) == 0
}

// Equal checks whether button sets are identical
func (bs ButtonSet) Equal(other ButtonSet) bool {
	if len(bs.rows) != len(other.rows) {
		return false
	}
	for idx := range bs.rows {
		if len(bs.rows[idx]) != len(other.rows[idx]) {
			return false
		}
		for jdx := range bs.rows[idx] {
			if bs.rows[idx][jdx] != other.rows[idx][jdx] {
				return false
			}
		}
	}
	return true
}
