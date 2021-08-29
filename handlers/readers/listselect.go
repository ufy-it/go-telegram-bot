package readers

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ufy-it/go-telegram-bot/handlers/buttons"
	"github.com/ufy-it/go-telegram-bot/logger"
)

// ListItem is a struct with wisible text and inner data
type ListItem struct {
	Text string
	Data string
}

// UserSelectedListReply contains a list of user-selected items
type UserSelectedListReply struct {
	Items []ListItem
	Data  string
	Exit  bool
}

const (
	prevPageButtonData = "__list_prev_page"
	nextPageButtonData = "__list_next_page"

	removeFilterCommand = "/remove_filter"
)

// SelectItemFromList asks user to select an item from the list
func SelectItemFromList(
	ctx context.Context,
	conversation BotConversation,
	text string,
	items []ListItem,
	pageSize int,
	navigation buttons.ButtonSet,
	prevPageText string,
	nextPageText string,
	filterText string,
	removeFilterText string) (UserTextAndDataReply, error) {

	filter := ""
	filterChanged := false
	startIndex := 0
	prevPageButton := buttons.NewButton(prevPageText, prevPageButtonData)
	nextPageButton := buttons.NewButton(nextPageText, nextPageButtonData)
	msgID, err := conversation.SendText(text)
	if err != nil {
		return UserTextAndDataReply{}, err
	}
	prevButtons := buttons.EmptyButtonSet()
	defer func() { //clear buttons after exit
		if filter != "" || !prevButtons.IsEmpty() {
			err := conversation.EditMessageText(msgID, text)
			if err != nil {
				logger.Warning("failed to edit message after list select: %v", err)
			}
		}
	}()

	for {
		bts := buttons.EmptyButtonSet()
		dataToText := make(map[string]string)
		textWithFilter := text
		if filter != "" {
			textWithFilter += fmt.Sprintf("\n\n <b>%s</b> \"%s\". (<i>%s: %s</i>)", filterText, filter, removeFilterText, removeFilterCommand)
		}
		filteredItems := make([]ListItem, 0)
		for _, item := range items {
			if filter == "" || strings.Contains(strings.ToLower(item.Text), strings.ToLower(filter)) {
				filteredItems = append(filteredItems, item)
			}
		}
		if len(filteredItems) > 0 {
			pageNavigation := make([]buttons.Button, 0)
			if startIndex > 0 {
				pageNavigation = append(pageNavigation, prevPageButton)
			}
			if startIndex+pageSize < len(filteredItems) {
				pageNavigation = append(pageNavigation, nextPageButton)
			}
			if len(pageNavigation) > 0 {
				bts = buttons.NewSingleRowButtonSet(pageNavigation...)
			}
			endSlice := startIndex + pageSize
			if endSlice > len(filteredItems) {
				endSlice = len(filteredItems)
			}
			for _, item := range filteredItems[startIndex:endSlice] {
				dataToText[item.Data] = item.Text
				bts = bts.Join(buttons.NewButtonRow(buttons.NewButton(item.Text, item.Data)))
			}
		}
		bts = bts.JoinSet(navigation)
		if !bts.Equal(prevButtons) || filterChanged {
			prevButtons = bts
			err := conversation.EditMessageTextAndInlineMarkup(msgID, textWithFilter, bts.GetInlineKeyboard())
			if err != nil {
				return UserTextAndDataReply{}, fmt.Errorf("failed to display list items: %v", err)
			}
		}

		filterChanged = false
		result := ReadRawTextAndDataResult(ctx, conversation)
		if result.Exit {
			return result, nil
		}
		if result.Data != "" {
			data, err := bts.FindButtonData(result.Data)
			if err != nil {
				logger.Warning("unknown button pressed during list select: %v", err)
			} else {
				switch data {
				case prevPageButtonData:
					startIndex -= pageSize
					if startIndex < 0 {
						startIndex = 0
					}
				case nextPageButtonData:
					startIndex += pageSize
				default:
					return UserTextAndDataReply{
						Text: dataToText[data],
						Data: data,
					}, nil
				}
			}
		}
		if result.Text != "" {
			if result.Text == removeFilterCommand {
				if filter != "" {
					filter = ""
					filterChanged = true
					startIndex = 0
				}
			} else {
				if filter != result.Text {
					filter = result.Text
					filterChanged = true
					startIndex = 0
				}
			}
			err = conversation.DeleteMessage(result.MessageID)
			if err != nil {
				return UserTextAndDataReply{}, fmt.Errorf("failed to delete user message: %v", err)
			}
		}
	}
}

// MultySelectItemFromList asks a user to select several items from the list
func MultySelectItemFromList(
	ctx context.Context,
	conversation BotConversation,
	text string,
	items []ListItem,
	selectedItems []ListItem,
	pageSize int,
	selectedText string,
	removeSelectedText string,
	navigation buttons.ButtonSet,
	prevPageText string,
	nextPageText string,
	filterText string,
	removeFilterText string) (UserSelectedListReply, error) {
	filter := ""
	filterChanged := false
	selectedChanged := false
	startIndex := 0
	removeItemRe := regexp.MustCompile(`/remove_\d+`)
	indexRe := regexp.MustCompile(`\d+`)
	prevPageButton := buttons.NewButton(prevPageText, prevPageButtonData)
	nextPageButton := buttons.NewButton(nextPageText, nextPageButtonData)
	msgID, err := conversation.SendText(text)
	if err != nil {
		return UserSelectedListReply{}, err
	}
	prevButtons := buttons.EmptyButtonSet()
	selected := UserSelectedListReply{
		Items: selectedItems,
		Data:  "",
		Exit:  false,
	}
	selectedData := make(map[string]struct{})
	defer func() { //clear buttons after exit
		if len(selected.Items) > 0 || filter != "" || !prevButtons.IsEmpty() {
			err := conversation.EditMessageText(msgID, text)
			if err != nil {
				logger.Warning("failed to edit message after multy select: %v", err)
			}
		}
	}()
	for {
		bts := buttons.EmptyButtonSet()
		dataToText := make(map[string]string)
		textWithFilter := text
		if len(selected.Items) > 0 {
			textWithFilter = fmt.Sprintf("%s\n\n%s\n", text, selectedText)
			for idx, item := range selected.Items {
				textWithFilter += fmt.Sprintf("%d. %s (%s /remove_%d)\n", idx+1, item.Text, removeSelectedText, idx+1)
			}
		}
		if filter != "" {
			textWithFilter += fmt.Sprintf("\n\n <b>%s</b> \"%s\". (<i>%s: %s</i>)", filterText, filter, removeFilterText, removeFilterCommand)
		}

		filteredItems := make([]ListItem, 0)
		for _, item := range items {
			if _, find := selectedData[item.Data]; !find && (filter == "" || strings.Contains(strings.ToLower(item.Text), strings.ToLower(filter))) {
				filteredItems = append(filteredItems, item)
			}
		}
		for {
			if len(filteredItems) <= startIndex && startIndex > 0 {
				startIndex -= pageSize
			} else {
				break
			}
		}
		if startIndex < 0 {
			startIndex = 0
		}
		if len(filteredItems) > 0 {
			pageNavigation := make([]buttons.Button, 0)
			if startIndex > 0 {
				pageNavigation = append(pageNavigation, prevPageButton)
			}
			if startIndex+pageSize < len(filteredItems) {
				pageNavigation = append(pageNavigation, nextPageButton)
			}
			if len(pageNavigation) > 0 {
				bts = buttons.NewSingleRowButtonSet(pageNavigation...)
			}
			endSlice := startIndex + pageSize
			if endSlice > len(filteredItems) {
				endSlice = len(filteredItems)
			}
			for _, item := range filteredItems[startIndex:endSlice] {
				dataToText[item.Data] = item.Text
				bts = bts.Join(buttons.NewButtonRow(buttons.NewButton(item.Text, item.Data)))
			}
		}
		bts = bts.JoinSet(navigation)
		if !bts.Equal(prevButtons) || filterChanged || selectedChanged {
			prevButtons = bts
			err := conversation.EditMessageTextAndInlineMarkup(msgID, textWithFilter, bts.GetInlineKeyboard())
			if err != nil {
				return UserSelectedListReply{}, fmt.Errorf("failed to display list items: %v", err)
			}
		}

		filterChanged = false
		selectedChanged = false
		result := ReadRawTextAndDataResult(ctx, conversation)
		if result.Exit {
			return UserSelectedListReply{Exit: true}, nil
		}
		if result.Data != "" {
			data, err := bts.FindButtonData(result.Data)
			if err != nil {
				logger.Warning("unknown button pressed during multy select: %v", err)
			} else {
				switch data {
				case prevPageButtonData:
					startIndex -= pageSize
				case nextPageButtonData:
					startIndex += pageSize
				default:
					if text, found := dataToText[data]; found {
						selectedChanged = true
						selected.Items = append(selected.Items, ListItem{Text: text, Data: data})
						selectedData[data] = struct{}{}
					} else {
						selected.Data = data
						return selected, nil
					}
				}
			}
		}
		if result.Text != "" {
			if removeItemRe.MatchString(result.Text) {
				indexes := indexRe.FindAllString(result.Text, 1)
				if len(indexes) < 1 {
					return UserSelectedListReply{}, fmt.Errorf("failed to parse index from command %s", result.Text)
				}
				index, err := strconv.Atoi(indexes[0])
				if err != nil {
					return UserSelectedListReply{}, fmt.Errorf("failed to parse int index from command %s", result.Text)
				}
				index--
				if index < 0 || index >= len(selected.Items) {
					logger.Warning("user asked to remove index %d that is out of range [0..%d)", index, len(selected.Items))
				} else {
					delete(selectedData, selected.Items[index].Data)
					copy(selected.Items[index:], selected.Items[index+1:])
					selected.Items = selected.Items[:len(selected.Items)-1]
					selectedChanged = true
				}
			} else if result.Text == removeFilterCommand {
				if filter != "" {
					filter = ""
					filterChanged = true
					startIndex = 0
				}
			} else {
				if filter != result.Text {
					filter = result.Text
					filterChanged = true
					startIndex = 0
				}
			}
			err = conversation.DeleteMessage(result.MessageID)
			if err != nil {
				return UserSelectedListReply{}, fmt.Errorf("failed to delete user message: %v", err)
			}
		}
	}
}
