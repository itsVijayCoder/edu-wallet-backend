package service

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/dto"
)

type ReceiptRenderer interface {
	RenderReceipt(ctx context.Context, receipt dto.ReceiptResponse) (*dto.ReceiptDownloadResponse, error)
}

type PDFReceiptRenderer struct{}

func NewPDFReceiptRenderer() *PDFReceiptRenderer {
	return &PDFReceiptRenderer{}
}

func (r *PDFReceiptRenderer) RenderReceipt(_ context.Context, receipt dto.ReceiptResponse) (*dto.ReceiptDownloadResponse, error) {
	lines := []string{
		"EduWallet Receipt",
		"Receipt: " + receipt.ReceiptNumber,
		"Date: " + receipt.IssueDate,
		"Student: " + receiptStudentName(receipt),
		"Payment Method: " + strings.ToUpper(receipt.PaymentMethod),
		"Amount: INR " + formatPaise(receipt.AmountPaise),
	}
	if len(receipt.Allocations) > 0 {
		lines = append(lines, "", "Invoice Allocations")
		for _, allocation := range receipt.Allocations {
			ref := allocation.InvoiceNumber
			if ref == "" {
				ref = allocation.InvoiceID.String()
			}
			lines = append(lines, ref+"  INR "+formatPaise(allocation.AmountPaise))
		}
	}

	pdf := buildSimplePDF(lines)
	return &dto.ReceiptDownloadResponse{
		Filename:    "receipt-" + receipt.ReceiptNumber + ".pdf",
		ContentType: "application/pdf",
		Bytes:       pdf,
	}, nil
}

func receiptStudentName(receipt dto.ReceiptResponse) string {
	if receipt.Student == nil {
		return receipt.StudentID.String()
	}
	return strings.TrimSpace(receipt.Student.FirstName + " " + receipt.Student.LastName)
}

func formatPaise(amount int64) string {
	rupees := amount / 100
	paise := amount % 100
	return fmt.Sprintf("%d.%02d", rupees, paise)
}

func buildSimplePDF(lines []string) []byte {
	var content bytes.Buffer
	content.WriteString("BT\n/F1 12 Tf\n50 780 Td\n")
	for i, line := range lines {
		if i > 0 {
			content.WriteString("0 -18 Td\n")
		}
		content.WriteString("(")
		content.WriteString(escapePDFText(line))
		content.WriteString(") Tj\n")
	}
	content.WriteString("ET\n")

	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		"<< /Length " + strconv.Itoa(content.Len()) + " >>\nstream\n" + content.String() + "endstream",
	}

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = out.Len()
		out.WriteString(strconv.Itoa(i + 1))
		out.WriteString(" 0 obj\n")
		out.WriteString(obj)
		out.WriteString("\nendobj\n")
	}
	xrefOffset := out.Len()
	out.WriteString("xref\n0 ")
	out.WriteString(strconv.Itoa(len(objects) + 1))
	out.WriteString("\n0000000000 65535 f \n")
	for i := 1; i < len(offsets); i++ {
		out.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	out.WriteString("trailer\n<< /Size ")
	out.WriteString(strconv.Itoa(len(objects) + 1))
	out.WriteString(" /Root 1 0 R >>\nstartxref\n")
	out.WriteString(strconv.Itoa(xrefOffset))
	out.WriteString("\n%%EOF\n")
	return out.Bytes()
}

func escapePDFText(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "(", `\(`)
	value = strings.ReplaceAll(value, ")", `\)`)
	return value
}
