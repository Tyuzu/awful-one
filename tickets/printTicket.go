package tickets

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/phpdave11/gofpdf"
	"github.com/skip2/go-qrcode"
)

const hmacSecret = "your-very-secret-key" // keep secure

// GenerateQRPayload returns a secure payload string: eventID|ticketID|uniqueCode|timestamp|signature
func GenerateQRPayload(eventID, ticketID, uniqueCode string) string {
	timestamp := time.Now().Unix()
	data := fmt.Sprintf("%s|%s|%s|%d", eventID, ticketID, uniqueCode, timestamp)

	h := hmac.New(sha256.New, []byte(hmacSecret))
	h.Write([]byte(data))
	sig := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return fmt.Sprintf("%s|%s", data, sig)
}

// PrintTicket generates a PDF ticket with QR code
func PrintTicket(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		eventID := ps.ByName("eventid")
		uniqueCode := r.URL.Query().Get("uniqueCode")

		tokenString := r.Header.Get("Authorization")
		claims, err := utils.ValidateJWT(tokenString)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if uniqueCode == "" {
			http.Error(w, "Unique code is required", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Fetch ticket via Database interface
		var ticket models.PurchasedTicket
		err = app.DB.FindOne(ctx, purchasedTicketsCollection, map[string]any{
			"eventid":    eventID,
			"uniquecode": uniqueCode,
		}, &ticket)
		if err != nil {
			http.Error(w, "Ticket not found", http.StatusNotFound)
			return
		}

		ticket.BuyerName = claims.Username

		// Generate QR payload
		qrPayload := GenerateQRPayload(ticket.EventID, ticket.TicketID, ticket.UniqueCode)
		qrPNG, _ := qrcode.Encode(qrPayload, qrcode.Medium, 256)

		// Create PDF
		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()

		// Header
		pdf.SetFont("Arial", "B", 24)
		pdf.SetTextColor(0, 102, 204)
		pdf.Cell(0, 15, "🎫 Event Ticket")
		pdf.Ln(20)

		// Event Info
		pdf.SetFont("Arial", "B", 14)
		pdf.SetTextColor(0, 0, 0)
		pdf.Cell(0, 8, "Event Information")
		pdf.Ln(10)

		pdf.SetFont("Arial", "", 12)
		pdf.Cell(0, 6, fmt.Sprintf("Event ID: %s", ticket.EventID))
		pdf.Ln(6)
		pdf.Cell(0, 6, fmt.Sprintf("Ticket ID: %s", ticket.TicketID))
		pdf.Ln(6)
		pdf.Cell(0, 6, fmt.Sprintf("Purchase Date: %s", ticket.PurchaseDate.Format("02 Jan 2006 15:04")))
		pdf.Ln(10)

		// Attendee Info
		pdf.SetFont("Arial", "B", 14)
		pdf.Cell(0, 8, "Attendee Information")
		pdf.Ln(10)

		pdf.SetFont("Arial", "", 12)
		pdf.Cell(0, 6, fmt.Sprintf("Name: %s", ticket.BuyerName))
		pdf.Ln(6)
		pdf.Cell(0, 6, fmt.Sprintf("Unique Code: %s", ticket.UniqueCode))
		pdf.Ln(12)

		// QR Code
		imageOpts := gofpdf.ImageOptions{ImageType: "PNG"}
		pdf.RegisterImageOptionsReader("qr", imageOpts, bytes.NewReader(qrPNG))
		pdf.ImageOptions("qr", 160, 40, 40, 40, false, imageOpts, 0, "")

		// Footer
		pdf.SetY(-30)
		pdf.SetFont("Arial", "I", 10)
		pdf.SetTextColor(128, 128, 128)
		pdf.Cell(0, 5, "This ticket is non-transferable except via official platform.")
		pdf.Ln(5)
		pdf.Cell(0, 5, "Valid only with QR code at entry.")

		// Output PDF
		var buf bytes.Buffer
		if err := pdf.Output(&buf); err != nil {
			http.Error(w, "Failed to generate PDF", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", "attachment; filename=ticket-"+ticket.UniqueCode+".pdf")
		w.WriteHeader(http.StatusOK)
		w.Write(buf.Bytes())
	}
}
