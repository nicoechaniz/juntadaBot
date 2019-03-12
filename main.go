package main

import (
	"fmt"
	"net/http"

	//"os"

	//"time"
	"io/ioutil"
	//"strconv"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var token string

var TelegramChatID string

//  *** Structured Variables  ***
type attendanceType struct {
	UserID    int
	FirstName string
	LastName  string
}

type eventStructType struct {
	Title     string
	Location  string
	Date      string
	Owner     int
	OwnerUN   string
	OwnerName string
	//OwnerLN   string
	isClosed bool
	Message  struct {
		MessageID int
	}
	Attending    []attendanceType
	NotAttending []attendanceType
}

var eventStruct [5]eventStructType

//UpdateType used for deconstructing the update message variables
type UpdateType struct {
	Ok     bool `json:"ok"`
	Result []struct {
		UpdateID int `json:"update_id"`
		Message  struct {
			MessageID int `json:"message_id"`
			From      struct {
				ID           int    `json:"id"`
				IsBot        bool   `json:"is_bot"`
				FirstName    string `json:"first_name"`
				LastName     string `json:"last_name"`
				Username     string `json:"username"`
				LanguageCode string `json:"language_code"`
			} `json:"from"`
			Chat struct {
				ID        int    `json:"id"`
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
				Title     string `json:"Title"`
				Username  string `json:"username"`
				Type      string `json:"type"`
			} `json:"chat"`
			Date     int         `json:"date"`
			Text     string      `json:"text"`
			Entities [1]struct { //Hard coded 1 to force the 0 entity to be created for line 73.  Done to avoid index out of bounds
				Offset int    `json:"offset"`
				Length int    `json:"length"`
				Type   string `json:"type"`
			} `json:"entities"`
		} `json:"message"`
		CallbackQuery struct {
			ID   string `json:"id"`
			From struct {
				ID           int    `json:"id"`
				IsBot        bool   `json:"is_bot"`
				FirstName    string `json:"first_name"`
				LastName     string `json:"last_name"`
				Username     string `json:"username"`
				LanguageCode string `json:"language_code"`
			} `json:"from"`
			Message struct {
				MessageID int `json:"message_id"`
				From      struct {
					ID        int    `json:"id"`
					IsBot     bool   `json:"is_bot"`
					FirstName string `json:"first_name"`
					Username  string `json:"username"`
				} `json:"from"`
				Chat struct {
					ID        int    `json:"id"`
					FirstName string `json:"first_name"`
					LastName  string `json:"last_name"`
					Username  string `json:"username"`
					Type      string `json:"type"`
				} `json:"chat"`
				Date     int    `json:"date"`
				Text     string `json:"text"`
				Entities []struct {
					Offset int    `json:"offset"`
					Length int    `json:"length"`
					Type   string `json:"type"`
				} `json:"entities"`
			} `json:"message"`
			ChatInstance string `json:"chat_instance"`
			Data         string `json:"data"`
		} `json:"callback_query"`
	} `json:"result"`
}

var Update UpdateType

type sentMessageType struct {
	Ok     bool `json:"ok"`
	Result struct {
		MessageID int `json:"message_id"`
		From      struct {
			ID        int    `json:"id"`
			IsBot     bool   `json:"is_bot"`
			FirstName string `json:"first_name"`
			Username  string `json:"username"`
		} `json:"from"`
		Chat struct {
			ID        int    `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Username  string `json:"username"`
			Type      string `json:"type"`
		} `json:"chat"`
		Date int    `json:"date"`
		Text string `json:"text"`
	} `json:"result"`
}

var sentMessage sentMessageType

var messageOffset struct {
	Offset int
}

var replacer = strings.NewReplacer("%", "%25", " ", "%20", "'", "%27", "+", "%2B", ".", "%2E", "/", "%2F", "#", "%23", "&", "%26", "*", "\\%2A", "_", "%5F", "`", "%60")
var userReplacer = strings.NewReplacer("*", "%2A", "#", "%23", "&", "%26", " ", "%20")

func clearAllEvents() {
	for event := range eventStruct {
		eventStruct[event].isClosed = true
	}
}

func main() {
	fmt.Println(token)
	fmt.Println(TelegramChatID)
	/*fmt.Println("-----CREATING SLICES-----")

	for event := range eventStruct {
		eventStruct[event].Attending = make([]struct {
			UserID    int
			FirstName string
			LastName  string
		}, 1, 50)
		eventStruct[event].NotAttending = make([]struct {
			UserID    int
			FirstName string
			LastName  string
		}, 1, 50)
	}*/

	fmt.Println("-----SETTING EVENTS TO CLOSED-----")
	http.Get(token + "getUpdates?offset=-1")
	clearAllEvents()
	fmt.Println("-----STARTING PROGRAM-----")
	for {
		if getUpdates() == true { //Gets and stores latest updates
			for i := range Update.Result {
				if Update.Result[i].Message.Entities[0].Type == "bot_command" && strconv.Itoa(Update.Result[i].Message.Chat.ID) == TelegramChatID && Update.Result[i].Message.From.IsBot == false && Update.Result[i].CallbackQuery.Data == "" {
					parseCommand(i)
				} else if Update.Result[i].CallbackQuery.Data != "" {
					fmt.Println("CALLBACK FUNC")
					for event := range eventStruct {
						if eventStruct[event].Message.MessageID == Update.Result[i].CallbackQuery.Message.MessageID {
							updateEvent(i, event)
							break
						}
					}
				}
				if i == len(Update.Result)-1 {
					fmt.Print("Updating Offset: ")
					messageOffset.Offset = Update.Result[i].UpdateID + 1 //Telegram API counts a message as 'read' when the getUpdates offset is 1 higher than the latest message sent.
					fmt.Println(messageOffset.Offset)
				}
			}
			if len(Update.Result) != 0 {
				clearAllResults()
			}
		}
	}
}

func getUpdates() bool {
	a, err := http.Get(token + "getUpdates?offset=" + strconv.Itoa(messageOffset.Offset) + "&timeout=15")
	if err != nil {
		fmt.Println("Unable to API Call. Trying again in 15 seconds...")
		fmt.Println(err)
		time.Sleep(15 * time.Second)
		return false
	}
	defer a.Body.Close()

	body, err := ioutil.ReadAll(a.Body) // Pulling data from Body into io.Reader variable

	if string(body) == `{"ok":true,"result":[]}` {
		return false
	}

	fmt.Println(string(body))
	//fmt.Println("-----DECODING-----")
	decoder := json.NewDecoder(strings.NewReader(string(body))) // New Decoder pulling data from body Reader
	decoder.Decode(&Update)                                     //Decode into variable

	return true
}

func parseCommand(i int) {
	var a []string
	if Update.Result[i].Message.Text != "" {
		fmt.Println(Update.Result[i].Message.Chat.Title + " - " + Update.Result[i].Message.Text)
	} else {
		return
	}
	expression, err := regexp.Compile("\\/make|\\/close") // Bot Command Regex
	if err != nil {
		fmt.Println("Unable to parse command.  Please try again later")
		fmt.Println(err)
		return
	}
	a = expression.FindStringSubmatch(Update.Result[i].Message.Text)
	if a != nil {
		switch a[0] {
		case "/make":
			fmt.Println("MAKE: ")
			for event := range eventStruct {
				if eventStruct[event].isClosed == true {
					fmt.Println("-----STARTING EVENT " + strconv.Itoa(event) + "-----")
					if createEvent(i, event) == false {
						return
					}
					fmt.Println("-----Event " + strconv.Itoa(event) + " Created-----")
					if pushEvent(event) == true {
						eventStruct[event].Message.MessageID = sentMessage.Result.MessageID
						fmt.Println("Event " + strconv.Itoa(event) + " Message ID " + strconv.Itoa(eventStruct[event].Message.MessageID))
						return
					} else {
						eventStruct[event].isClosed = true
						return
					}
				}
			}
			fmt.Println("ERROR: Unable to make event(max limit reached)")
			http.Get((token + `sendMessage?chat_id=` + TelegramChatID + `&text=Unable to add event: too many events open! Only 5 allowed at a time.`))
		case "/close":
			fmt.Println("CLOSE: ")
			for event := range eventStruct {
				if eventStruct[event].Owner == Update.Result[i].Message.From.ID {
					if closeEvent(i, event) == true {
						eventStruct[event].isClosed = true
						return
					}
				}
			}
		/*case "/location":
		locEvent(i)*/
		default:
			fmt.Println("Failed to parseCommand case/switch.")
		}
	}
}

func createEvent(i int, event int) bool {
	var a []string
	fmt.Println(Update.Result[i].Message.Chat.Title + " - " + Update.Result[i].Message.Text)
	expression, _ := regexp.Compile(" (.+)\\@ ?(.+) ([oOnN]{2} ((([0-9]{1,2})\\/([0-9]{1,2})\\/([0-9]{2,4})),? (([0-9]{1,2}):([0-9]{2}) (A|P)M)))")
	//expression := regexp.MustCompile(" (.+)\\@ ?(.+?(?=on|at)|.+) ?(at)?(on (([0-9]{2})\\/([0-9]{2})\\/([0-9]{2,4})))? ?(([0-9]{1,2}):?([0-9]{2})? ?(A|P)M)?") // Title @ Location <on Date> <Time>
	a = expression.FindStringSubmatch(Update.Result[i].Message.Text)
	fmt.Println(a)
	if a == nil {
		fmt.Println("empty make")
		http.Get(token + "sendMessage?chat_id=" + TelegramChatID + "&parse_mode=Markdown&text=Unable%20to%20create%20event.%20Please%20use%20the%20following%20template%3A%0A*/make%20Title%20%40%20location%20address%20on%20MM/DD/YY%20HH%3AMM%20PM*%0AEverything%20is%20Mandatory.")
		return false
	}

	for group := range a {
		if a[group] != "" {
			fmt.Println(a[group])
		}
	}

	fmt.Println(a)
	eventStruct[event].Title = a[1]
	fmt.Println("Event Title = " + eventStruct[event].Title)
	eventStruct[event].Location = a[2]
	fmt.Println("Event Location = " + eventStruct[event].Location)
	eventStruct[event].Date = a[4]
	fmt.Println("Event Date/Time = " + eventStruct[event].Date)

	fmt.Println("Setting event owner to @" + Update.Result[i].Message.From.Username + ", " + strconv.Itoa(Update.Result[i].Message.From.ID))
	eventStruct[event].Owner = Update.Result[i].Message.From.ID
	eventStruct[event].OwnerUN = Update.Result[i].Message.From.Username
	eventStruct[event].OwnerName = Update.Result[i].Message.From.FirstName + " " + Update.Result[i].Message.From.LastName
	fmt.Println("Setting event " + strconv.Itoa(event) + " to the sent message")
	eventStruct[event].Message.MessageID = Update.Result[i].Message.MessageID
	fmt.Println("Setting event " + strconv.Itoa(event) + " to OPEN")
	eventStruct[event].isClosed = false

	return true
}

//sendMessage?chat_id=176785082&text=Hello&reply_markup={"inline_keyboard":[[{"text":"Yest Test","callback_data":"yes"}]]}

func pushEvent(event int) bool {
	fmt.Println("Sending RSVP post")
	fmt.Println(token + `sendMessage?chat_id=` + TelegramChatID + `&text=*RSVP Event ` + strconv.Itoa(event+1) + ` for ` + replacer.Replace(eventStruct[event].Title) + `*%0AHost:%20` + eventStruct[event].OwnerName + `  (@` + eventStruct[event].OwnerUN + `)%0ALocation: ` + replacer.Replace(eventStruct[event].Location) + `%0A*Date:*%20` + eventStruct[event].Date + `%0A%0A*Going:*%0A%0A*Not Going:*%0A%0ARSVP Below.&parse_mode=Markdown&reply_markup={"inline_keyboard":[[{"text":"✅","callback_data":"Go"},{"text":"✅ %2B1","callback_data":"%2B1"},{"text":"❌","callback_data":"No"}]]}`)
	a, err := http.Get(token + `sendMessage?chat_id=` + TelegramChatID + `&text=*RSVP Event ` + strconv.Itoa(event+1) + ` for ` + replacer.Replace(eventStruct[event].Title) + `*%0AHost:%20` + userReplacer.Replace(eventStruct[event].OwnerName) + `  (@` + eventStruct[event].OwnerUN + `)%0ALocation: ` + replacer.Replace(eventStruct[event].Location) + `%0A*Date:*%20` + eventStruct[event].Date + `%0A%0A*Going:*%0A%0A*Not Going:*%0A%0ARSVP Below.&parse_mode=Markdown&reply_markup={"inline_keyboard":[[{"text":"✅","callback_data":"Go"},{"text":"✅ %2B1","callback_data":"%2B1"},{"text":"❌","callback_data":"No"}]]}`)
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer a.Body.Close()
	body, err := ioutil.ReadAll(a.Body) // Pulling data from Body into io.Reader variable
	if string(body) == `{"ok":true,"result":[]}` {
		return false
	} else if a.StatusCode != http.StatusOK {
		fmt.Println("Unable to update the event.  Please check if there are any outlying special characters that may be affecting the event.")
		fmt.Println("-----SETTING EVENT TO CLOSED-----")
		eventStruct[event].isClosed = true
		return false
	}
	fmt.Println(string(body))
	//fmt.Println("-----DECODING UPDATE-----")
	decoder := json.NewDecoder(strings.NewReader(string(body))) // New Decoder pulling data from body Reader
	decoder.Decode(&sentMessage)                                //Decode into variable

	return true
}

func updateEvent(i int, event int) {
	if eventStruct[event].isClosed == true {
		fmt.Println("Event Closed. Unable to add " + Update.Result[i].CallbackQuery.From.FirstName + " to event " + eventStruct[event].Title)
		return
	}

	var attendingNames string
	var notAttendingNames string

	fmt.Println("Updating RSVP for " + Update.Result[i].CallbackQuery.From.FirstName + Update.Result[i].CallbackQuery.From.LastName + ", User ID " + strconv.Itoa(Update.Result[i].CallbackQuery.From.ID) + " on Event " + eventStruct[event].Title + " " + strconv.Itoa(eventStruct[event].Message.MessageID))

	for a := range eventStruct[event].Attending {
		if len(eventStruct[event].Attending) == 0 {
			break
		}
		if eventStruct[event].Attending[a].UserID == Update.Result[i].CallbackQuery.From.ID {
			eventStruct[event].Attending = append(eventStruct[event].Attending[:a], eventStruct[event].Attending[a+1:]...)
			break
		}
	}
	for a := range eventStruct[event].NotAttending {
		if len(eventStruct[event].NotAttending) == 0 {
			break
		}
		if eventStruct[event].NotAttending[a].UserID == Update.Result[i].CallbackQuery.From.ID {
			eventStruct[event].NotAttending = append(eventStruct[event].NotAttending[:a], eventStruct[event].NotAttending[a+1:]...)
			break
		}
	}

	switch Update.Result[i].CallbackQuery.Data {
	case "Go":
		if Update.Result[i].CallbackQuery.From.LastName == "" {
			eventStruct[event].Attending = append(eventStruct[event].Attending, attendanceType{Update.Result[i].CallbackQuery.From.ID, Update.Result[i].CallbackQuery.From.FirstName, " "})
		} else {
			eventStruct[event].Attending = append(eventStruct[event].Attending, attendanceType{Update.Result[i].CallbackQuery.From.ID, Update.Result[i].CallbackQuery.From.FirstName, Update.Result[i].CallbackQuery.From.LastName})
		}
	case "+1":
		if Update.Result[i].CallbackQuery.From.LastName == "" {
			eventStruct[event].Attending = append(eventStruct[event].Attending, attendanceType{Update.Result[i].CallbackQuery.From.ID, Update.Result[i].CallbackQuery.From.FirstName, " %2B 1"})
		} else {
			eventStruct[event].Attending = append(eventStruct[event].Attending, attendanceType{Update.Result[i].CallbackQuery.From.ID, Update.Result[i].CallbackQuery.From.FirstName, Update.Result[i].CallbackQuery.From.LastName + " %2B 1"})
		}
	case "No":
		if Update.Result[i].CallbackQuery.From.LastName == "" {
			eventStruct[event].NotAttending = append(eventStruct[event].NotAttending, attendanceType{Update.Result[i].CallbackQuery.From.ID, Update.Result[i].CallbackQuery.From.FirstName, " "})
		} else {
			eventStruct[event].NotAttending = append(eventStruct[event].NotAttending, attendanceType{Update.Result[i].CallbackQuery.From.ID, Update.Result[i].CallbackQuery.From.FirstName, Update.Result[i].CallbackQuery.From.LastName})
		}
	}

	//eventStruct[event].Attending = append(eventStruct[event].Attending, attendanceType{Update.Result[i].CallbackQuery.From.ID, Update.Result[i].CallbackQuery.From.FirstName, Update.Result[i].CallbackQuery.From.LastName})

	for x := range eventStruct[event].Attending {
		attendingNames = attendingNames + strconv.Itoa(x+1) + ") " + eventStruct[event].Attending[x].FirstName + " " + eventStruct[event].Attending[x].LastName + "%0A"
	}
	for x := range eventStruct[event].NotAttending {
		notAttendingNames = notAttendingNames + strconv.Itoa(x+1) + ") " + eventStruct[event].NotAttending[x].FirstName + " " + eventStruct[event].NotAttending[x].LastName + "%0A"
	}
	fmt.Println(token + "editMessageText?chat_id=" + TelegramChatID + "&message_id=" + strconv.Itoa(eventStruct[event].Message.MessageID) + `&parse_mode=Markdown&text=*RSVP%20Event%20` + strconv.Itoa(event+1) + `%20for%20` + replacer.Replace(eventStruct[event].Title) + `%0AHost%3A%20` + eventStruct[event].OwnerName + `%20%20(@` + eventStruct[event].OwnerUN + `)%0ALocation:%20` + replacer.Replace(eventStruct[event].Location) + `%0A*Date:*%20` + eventStruct[event].Date + `%0A%0A*Going:*%0A` + replacer.Replace(attendingNames) + `%0A*Not%20Going:*%0A` + replacer.Replace(notAttendingNames) + `%0ARSVP%20Below.&reply_markup={"inline_keyboard":[[{"text":"✅","callback_data":"Go"},{"text":"✅ %2B1","callback_data":"%2B1"},{"text":"❌","callback_data":"No"}]]}`)
	editMessage, _ := http.Get(token + "editMessageText?chat_id=" + TelegramChatID + "&message_id=" + strconv.Itoa(eventStruct[event].Message.MessageID) + `&parse_mode=Markdown&text=*RSVP%20Event%20` + strconv.Itoa(event+1) + `%20for%20` + replacer.Replace(eventStruct[event].Title) + `*%0AHost%3A%20` + userReplacer.Replace(eventStruct[event].OwnerName) + `%20%20(@` + eventStruct[event].OwnerUN + `)%0ALocation:%20` + replacer.Replace(eventStruct[event].Location) + `%0A*Date:*%20` + eventStruct[event].Date + `%0A%0A*Going:*%0A` + userReplacer.Replace(attendingNames) + `%0A*Not%20Going:*%0A` + userReplacer.Replace(notAttendingNames) + `%0ARSVP%20Below.&reply_markup={"inline_keyboard":[[{"text":"✅","callback_data":"Go"},{"text":"✅ %2B1","callback_data":"%2B1"},{"text":"❌","callback_data":"No"}]]}`)
	readEditMessage, _ := ioutil.ReadAll(editMessage.Body)
	fmt.Println(string(readEditMessage))
}

func closeEvent(i int, event int) bool {
	var a []string

	expression, _ := regexp.Compile(" (\\d)")
	a = expression.FindStringSubmatch(Update.Result[i].Message.Text)
	fmt.Println(a)

	if strconv.Itoa(event+1) != a[1] {
		return false
	}

	var attendingNames string
	var notAttendingNames string

	for x := range eventStruct[event].Attending {
		attendingNames = attendingNames + strconv.Itoa(x+1) + ") " + eventStruct[event].Attending[x].FirstName + " " + eventStruct[event].Attending[x].LastName + "%0A"
	}
	for x := range eventStruct[event].NotAttending {
		notAttendingNames = notAttendingNames + strconv.Itoa(x+1) + ") " + eventStruct[event].NotAttending[x].FirstName + " " + eventStruct[event].NotAttending[x].LastName + "%0A"
	}
	//fmt.Println(token + "editMessageText?chat_id=" + TelegramChatID + "&message_id=" + strconv.Itoa(eventStruct[event].Message.MessageID) + `&parse_mode=Markdown&text=*RSVP%20Event%20` + strconv.Itoa(event+1) + `%20for%20` + replacer.Replace(eventStruct[event].Title) + `%0AHost%3A%20` + eventStruct[event].OwnerName + `%20%20(@` + eventStruct[event].OwnerUN + `)%0ALocation:%20` + replacer.Replace(eventStruct[event].Location) + `%0A*Date:*%20` + eventStruct[event].Date + `%0A%0A*Going:*%0A` + replacer.Replace(attendingNames) + `%0A*Not%20Going:*%0A` + replacer.Replace(notAttendingNames) + `%0ARSVP%20Below.&reply_markup={"inline_keyboard":[[{"text":"✅","callback_data":"Go"},{"text":"✅ %2B1","callback_data":"%2B1"},{"text":"❌","callback_data":"No"}]]}`)
	editMessage, _ := http.Get(token + "editMessageText?chat_id=" + TelegramChatID + "&message_id=" + strconv.Itoa(eventStruct[event].Message.MessageID) + `&parse_mode=Markdown&text=*CLOSED!*%0A*RSVP%20Event%20` + strconv.Itoa(event+1) + `%20for%20` + replacer.Replace(eventStruct[event].Title) + `*%0AHost%3A%20` + userReplacer.Replace(eventStruct[event].OwnerName) + `%20%20(@` + eventStruct[event].OwnerUN + `)%0ALocation:%20` + replacer.Replace(eventStruct[event].Location) + `%0A*Date:*%20` + eventStruct[event].Date + `%0A%0A*Going:*%0A` + userReplacer.Replace(attendingNames) + `%0A*Not%20Going:*%0A` + userReplacer.Replace(notAttendingNames) + `%0A*RSVP%20CLOSED*&reply_markup={"inline_keyboard":[[{"text":"✅","callback_data":"Go"},{"text":"✅ %2B1","callback_data":"%2B1"},{"text":"❌","callback_data":"No"}]]}`)
	readEditMessage, _ := ioutil.ReadAll(editMessage.Body)
	fmt.Println(string(readEditMessage))

	sendClose, _ := http.Get(token + "sendMessage?chat_id=" + TelegramChatID + `&parse_mode=Markdown&text=*CLOSED!*%0A*RSVP%20Event%20` + strconv.Itoa(event+1) + `%20for%20` + replacer.Replace(eventStruct[event].Title) + `*%0AHost%3A%20` + userReplacer.Replace(eventStruct[event].OwnerName) + `%20%20(@` + eventStruct[event].OwnerUN + `)%0ALocation:%20` + replacer.Replace(eventStruct[event].Location) + `%0A*Date:*%20` + eventStruct[event].Date + `%0A%0A*Going:*%0A` + userReplacer.Replace(attendingNames) + `%0A*Not%20Going:*%0A` + userReplacer.Replace(notAttendingNames))
	readSentClose, _ := ioutil.ReadAll(sendClose.Body)
	fmt.Println(string(readSentClose))

	eventStruct[event].Attending = nil
	eventStruct[event].NotAttending = nil
	return true
}

func clearAllResults() {
	fmt.Println("-----CLEARING RESULTS-----")
	Update.Result = nil
	//Update.Result = append(Update.Result[:len(Update.Result)-1])
	if len(Update.Result) == 0 {
		fmt.Println("All Results Cleared")
	}
}

/*func parseDate(month string, day string, year string) {
	mi, err := strconv.Atoi(month)
	if err != nil {
		fmt.Println("Unable to convert month to int")
		return
	}
	di, err := strconv.Atoi(day)
	if err != nil {
		fmt.Println("Unable to convert day to int")
		return
	}
	yi, err := strconv.Atoi(year)
	if err != nil {
		fmt.Println("Unable to convert year to int")
		return
	}

	m := time.Month(mi)
}*/
