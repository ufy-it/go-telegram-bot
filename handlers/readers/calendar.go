package readers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ufy-it/go-telegram-bot/handlers/buttons"
	"github.com/ufy-it/go-telegram-bot/logger"
)

type CalendarMode int

const (
	YearMode CalendarMode = iota
	MonthMode
	DayMode
)

const (
	yearsMode = "years"
	yearLable = "year"

	monthsMode = "months"
	monthLable = "month"

	dayLable = "day"

	prevPageLable = "prev"
	nextPageLable = "next"
)

const (
	yearPage     = 15
	yearHalfPage = 15/2 + 1
)

// AskReplyCalendarDate asks a user to select date between minDate and maxDate in calendar widget
func AskReplyCalendarDate(ctx context.Context, conversation BotConversation,
	text string,
	minDate time.Time,
	maxDate time.Time,
	currentDate time.Time,
	calendarMode CalendarMode,
	navigation buttons.ButtonSet,
	months [12]string,
	weekDays [7]string,
	prevPageText string,
	nextPageText string,
	location *time.Location) (UserTimeAndDataReply, error) {
	changed := true
	msgID, err := conversation.SendGeneralMessageWithKeyboardRemoveOnExit(conversation.NewMessage(text))
	if err != nil {
		return UserTimeAndDataReply{}, err
	}

	defer func() {
		if !minDate.After(maxDate) {
			err := conversation.RemoveReplyMarkup(msgID)
			if err != nil {
				logger.Warning("failed to remove reply makup in calendar widget: %v", err)
			}
		}
	}()

	buildDate := func(year int, month int, day int) time.Time {
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, location)
	}
	newDayButton := func(day int) buttons.Button {
		return buttons.NewButton(strconv.Itoa(day), fmt.Sprintf("%s%d", dayLable, day))
	}
	newMonthButton := func(index int) buttons.Button {
		return buttons.NewButton(months[index], fmt.Sprintf("%s%d", monthLable, index+1))
	}
	newYearButton := func(year int) buttons.Button {
		return buttons.NewButton(strconv.Itoa(year), fmt.Sprintf("%s%d", yearLable, year))
	}
	newPrevPageButton := func() buttons.Button {
		return buttons.NewButton(prevPageText, prevPageLable)
	}
	newNextPageButton := func() buttons.Button {
		return buttons.NewButton(nextPageText, nextPageLable)
	}
	newMonthModeNavigationRow := func() buttons.ButtonRow {
		row := buttons.NewButtonRow()
		if currentDate.Year() > minDate.Year() {
			row = append(row, newPrevPageButton())
		} else {
			row = append(row, buttons.NewEmptyIgnoreButton())
		}
		row = append(row, buttons.NewButton(strconv.Itoa(currentDate.Year()), yearsMode))
		if currentDate.Year() < maxDate.Year() {
			row = append(row, newNextPageButton())
		} else {
			row = append(row, buttons.NewEmptyIgnoreButton())
		}
		return row
	}
	newYearModeNavigationRow := func() buttons.ButtonRow {
		row := buttons.NewButtonRow()
		if currentDate.Year() > minDate.Year() {
			row = append(row, newPrevPageButton())
		} else {
			row = append(row, buttons.NewEmptyIgnoreButton())
		}
		if currentDate.Year()+yearPage <= maxDate.Year() {
			row = append(row, newNextPageButton())
		} else {
			row = append(row, buttons.NewEmptyIgnoreButton())
		}
		return row
	}
	newDayModeNavigationRow := func() buttons.ButtonRow {
		row := buttons.NewButtonRow()
		if buildDate(currentDate.Year(), int(currentDate.Month()), 1).After(minDate) {
			row = append(row, newPrevPageButton())
		} else {
			row = append(row, buttons.NewEmptyIgnoreButton())
		}
		row = append(row, buttons.NewButton(fmt.Sprintf("%s %d", months[int(currentDate.Month())-1], currentDate.Year()), monthsMode))
		if maxDate.After(buildDate(currentDate.Year(), int(currentDate.Month()), 1).AddDate(0, 1, -1)) {
			row = append(row, newNextPageButton())
		} else {
			row = append(row, buttons.NewEmptyIgnoreButton())
		}
		return row
	}
	var bs buttons.ButtonSet
	for {
		if minDate.After(currentDate) {
			currentDate = minDate
		}
		if currentDate.After(maxDate) {
			currentDate = maxDate
		}
		if changed {
			bs = buttons.EmptyButtonSet()
			switch calendarMode {
			case YearMode:
				navRow := newYearModeNavigationRow()
				if len(navRow) > 0 {
					bs = bs.Join(navRow)
				}
				row := buttons.NewButtonRow()
				for year := currentDate.Year(); year < currentDate.Year()+yearPage && year <= maxDate.Year(); year++ {
					row = append(row, newYearButton(year))
					if len(row) == 3 {
						bs = bs.Join(row)
						row = buttons.NewButtonRow()
					}

				}
				if len(row) > 0 {
					row = append(row, buttons.NewEmptyIgnoreButton())
					if len(row) < 3 {
						row = append(row, buttons.NewEmptyIgnoreButton())
					}
					bs = bs.Join(row)
				}
			case MonthMode:
				bs = bs.Join(newMonthModeNavigationRow())
				for rowIndex := 0; rowIndex < 4; rowIndex++ {
					row := buttons.NewButtonRow()
					for columnIndex := 0; columnIndex < 3; columnIndex++ {
						monthIndex := rowIndex*3 + columnIndex
						monthDate := buildDate(currentDate.Year(), monthIndex+1, 1)
						if maxDate.After(monthDate) && monthDate.AddDate(0, 1, -1).After(minDate) {
							row = append(row, newMonthButton(monthIndex))
						} else {
							row = append(row, buttons.NewEmptyIgnoreButton())
						}
					}
					bs = bs.Join(row)
				}
			case DayMode:
				bs = bs.Join((newDayModeNavigationRow()))
				row := buttons.NewButtonRow()
				for _, weekDay := range weekDays {
					row = append(row, buttons.NewIgnoreButton(weekDay))
				}
				bs = bs.Join(row)
				day := buildDate(currentDate.Year(), int(currentDate.Month()), 1)
				row = buttons.NewButtonRow()
				for i := 0; i < (int(day.Weekday())+6)%7; i++ {
					row = append(row, buttons.NewEmptyIgnoreButton())
				}
				lastDay := day.AddDate(0, 1, -1)
				for ; !day.After(lastDay); day = day.AddDate(0, 0, 1) {
					if !minDate.After(day) && !day.After(maxDate) {
						row = append(row, newDayButton(day.Day()))
					} else {
						row = append(row, buttons.NewEmptyIgnoreButton())
					}
					if len(row) == 7 {
						bs = bs.Join(row)
						row = buttons.NewButtonRow()
					}
				}
				if len(row) > 0 {
					for ; len(row) < 7; row = append(row, buttons.NewEmptyIgnoreButton()) {
					}
					bs = bs.Join(row)
				}
			}
			bs = bs.JoinSet(navigation)
			err := conversation.EditReplyMarkup(msgID, bs.GetInlineKeyboard())
			if err != nil {
				return UserTimeAndDataReply{}, err
			}
		}
		changed = false
		reply := ReadRawTextAndDataResult(ctx, conversation)
		if reply.Exit {
			return UserTimeAndDataReply{Exit: true}, nil
		}
		if reply.Data != "" {
			reply.Data, err = bs.FindButtonData(reply.Data)
			if err != nil {
				logger.Warning("unknown button pressed in callendar widget: %v", err)
				continue
			}
			changed = true
			switch reply.Data {
			case buttons.IgnoreButtonData:
				changed = false
				err = conversation.AnswerButton(reply.CallbackQueryID)
				if err != nil {
					return UserTimeAndDataReply{}, err
				}
			case yearsMode:
				calendarMode = YearMode
			case monthsMode:
				calendarMode = MonthMode
			case prevPageLable:
				switch calendarMode {
				case YearMode:
					currentDate = currentDate.AddDate(-yearHalfPage, 0, 0)
				case MonthMode:
					currentDate = currentDate.AddDate(-1, 0, 0)
				case DayMode:
					currentDate = currentDate.AddDate(0, -1, 0)
				default:
					return UserTimeAndDataReply{}, fmt.Errorf("this should never happen: cannot process callendar mode %d", calendarMode)
				}
			case nextPageLable:
				switch calendarMode {
				case YearMode:
					currentDate = currentDate.AddDate(yearHalfPage, 0, 0)
				case MonthMode:
					currentDate = currentDate.AddDate(1, 0, 0)
				case DayMode:
					currentDate = currentDate.AddDate(0, 1, 0)
				default:
					return UserTimeAndDataReply{}, fmt.Errorf("this should never happen: cannot process callendar mode %d", calendarMode)
				}
			default:
				if strings.HasPrefix(reply.Data, yearLable) {
					year, err := strconv.Atoi(strings.TrimPrefix(reply.Data, yearLable))
					if err != nil {
						return UserTimeAndDataReply{}, fmt.Errorf("failed to parse year from callback data %s: %v", reply.Data, err)
					}
					currentDate = buildDate(year, 1, 1)
					calendarMode = MonthMode
				} else if strings.HasPrefix(reply.Data, monthLable) {
					month, err := strconv.Atoi(strings.TrimPrefix(reply.Data, monthLable))
					if err != nil {
						return UserTimeAndDataReply{}, fmt.Errorf("failed to parse month from callback data %s: %v", reply.Data, err)
					}
					currentDate = buildDate(currentDate.Year(), month, 1)
					calendarMode = DayMode
				} else if strings.HasPrefix(reply.Data, dayLable) {
					day, err := strconv.Atoi(strings.TrimPrefix(reply.Data, dayLable))
					if err != nil {
						return UserTimeAndDataReply{}, fmt.Errorf("failed to parse day from callback data %s: %v", reply.Data, err)
					}
					return UserTimeAndDataReply{
						Data:            reply.Data,
						CallbackQueryID: reply.CallbackQueryID,
						Time:            buildDate(currentDate.Year(), int(currentDate.Month()), day),
					}, nil
				} else {
					return UserTimeAndDataReply{
						Data:            reply.Data,
						CallbackQueryID: reply.CallbackQueryID,
						Time:            currentDate,
					}, nil
				}
			}
		}
		if reply.Text != "" {
			err := conversation.DeleteMessage(reply.MessageID)
			if err != nil {
				return UserTimeAndDataReply{}, fmt.Errorf("failed to clear user's text-message: %v", err)
			}
		}
	}
}
