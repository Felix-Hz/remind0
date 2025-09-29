package app

import (
	"fmt"
	"log"
	"strings"
	"time"

	. "remind0/db"

	telegramClient "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

/**
 * Get updates using long-polling.
 * This will return a channel for updates.
 * Updates will be polled every 60 seconds.
 */
func ConnectBot(bot *telegramClient.BotAPI, offset Offset) telegramClient.UpdatesChannel {
	u := telegramClient.NewUpdate(offset.Offset)
	u.Timeout = 60
	log.Println("✅ Channel opened")
	return bot.GetUpdatesChan(u)
}

func HandleTelegramMessage(bot *telegramClient.BotAPI, update telegramClient.Update) {

	tgUserID := update.Message.Chat.ID                    // Get Telegram user ID
	body := update.Message.Text                           // Extract message text
	timestamp := time.Unix(int64(update.Message.Date), 0) // Extract timestamp
	username := update.Message.From.UserName              // Extract username
	firstName := update.Message.From.FirstName            // Extract first name
	lastName := update.Message.From.LastName              // Extract last name

	log.Printf("✅ Received message: %+v", struct {
		User      string
		Body      string
		Timestamp time.Time
	}{
		Body:      body,
		Timestamp: timestamp,
		User:      firstName + " " + lastName,
	})

	/**
	 * Validate the message: non-empty and within length limits (160 chars).
	 */
	if !validateMessage(body) {
		bot.Send(telegramClient.NewMessage(tgUserID, "⚠️ Message cannot be empty or exceed 160 characters."))
		return
	}

	/**
	 * Validate or create user.
	 */
	var user User
	result := DBClient.Where("user_id = ?", tgUserID).First(&user)
	if result.Error != nil {
		user = User{Username: username, UserID: tgUserID, FirstName: firstName, LastName: lastName}
		create := DBClient.Create(&user)
		if create.Error != nil {
			log.Printf("⚠️ Error creating user: %s", create.Error)
			bot.Send(telegramClient.NewMessage(tgUserID, "⚠️ Failed to create user profile. Please try again later."))
			return
		}
		log.Printf("✅ Created new user: %s (%d)", firstName+" "+lastName, tgUserID)
	}

	/**
	 * An exclamation mark indicates a user's wish to interact with the system.
	 */
	if strings.HasPrefix(body, "!") {
		r := dispatch(strings.TrimPrefix(body, "!"), user.ID)
		if r.Error != nil {
			log.Printf("⚠️ Error processing command: %s", r.Error)
			bot.Send(telegramClient.NewMessage(tgUserID, fmt.Sprintf("⚠️ Failed to process command: %s", r.UserError)))
			return
		}

		// TODO - Refactor to handle different command success messages.
		msg := "✅ Command executed successfully."
		if tx := r.Transaction; tx != nil {
			// Only removals are implemented so far.
			msg = successMessage(false, tx.ID, tx.Category, tx.Amount, tx.Notes, tx.Timestamp)
		}

		bot.Send(telegramClient.NewMessage(tgUserID, msg))
		return
	}

	/**
	 * Process incoming message.
	 */
	category, amount, notes, parseErr := processTx(body)
	if parseErr != nil {
		log.Printf("⚠️ Error parsing message: %s", parseErr)
		bot.Send(telegramClient.NewMessage(tgUserID, invalidMessageError()))
		return
	}

	/**
	 * Write message hash to prevent duplicates.
	 */
	hash := generateMessageHash(body, timestamp)

	/**
	 * Validate transaction uniqueness.
	 */
	var existingExpense Transaction
	result = DBClient.Where("hash = ?", hash).First(&existingExpense)
	if result.Error == nil {
		bot.Send(telegramClient.NewMessage(tgUserID, "⚠️ This expense was already recorded."))
		return
	}

	/**
	 * Persist transaction
	 */
	expense := Transaction{
		UserID:    user.ID,
		Category:  category,
		Amount:    amount,
		Notes:     notes,
		Timestamp: timestamp,
		Hash:      hash,
	}

	createTx := DBClient.Create(&expense)
	if createTx.Error != nil {
		log.Printf("⚠️ Error creating transaction: %s", createTx.Error)
		bot.Send(telegramClient.NewMessage(tgUserID, "⚠️ Failed to record expense. Please try again later."))
		return
	}

	/**
	 * Send confirmation message to user.
	 */
	bot.Send(telegramClient.NewMessage(tgUserID, successMessage(true, expense.ID, category, amount, notes, timestamp)))

	log.Printf("✅ Expense recorded: %+v", expense)
}
