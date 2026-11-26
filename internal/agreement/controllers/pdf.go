package controllers

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"lead-crm/backend/database"
	leadmodels "lead-crm/backend/internal/lead/models"
	"lead-crm/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-pdf/fpdf"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	pageMarginL  = 18.0
	pageMarginR  = 18.0
	pageContentW = 210.0 - pageMarginL - pageMarginR // 174mm
	pageTopY     = 34.0
)

var (
	durationDigitsRe  = regexp.MustCompile(`(\d+)`)
	numberedSectionRe = regexp.MustCompile(`^\d+\.\s+[A-Z]`)
	subItemRe         = regexp.MustCompile(`^([a-z]\.\s+)(.*)$`)
)

type rgbColor struct {
	r, g, b int
}

var (
	colorSlate900 = rgbColor{15, 23, 42}
	colorSlate800 = rgbColor{30, 41, 59}
	colorSlate600 = rgbColor{71, 85, 105}
	colorSlate500 = rgbColor{100, 116, 139}
	colorGreen700 = rgbColor{21, 128, 61}
	colorGreenHdr = rgbColor{53, 122, 87}
	colorBorder   = rgbColor{203, 213, 225}
	colorRowTint  = rgbColor{248, 250, 252}
	colorSlateHdr = rgbColor{241, 245, 249}
)

type textSegment struct {
	text  string
	bold  bool
	color *rgbColor
}

type pdfFontConfig struct {
	family  string
	unicode bool
}

func resolveAssetPath(rel string) string {
	candidates := []string{
		filepath.Join("assets", rel),
		filepath.Join("backend", "assets", rel),
		filepath.Join("..", "assets", rel),
		filepath.Join("..", "..", "assets", rel),
		filepath.Join("..", "..", "..", "assets", rel),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func resolveLogoPath() string {
	if p := resolveAssetPath("settlexpert-logo.png"); p != "" {
		return p
	}
	for _, p := range []string{"../frontend/public/logo.png", "frontend/public/logo.png"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func configurePDFFonts(pdf *fpdf.Fpdf) pdfFontConfig {
	regular := resolveAssetPath(filepath.Join("fonts", "Inter-Regular.ttf"))
	bold := resolveAssetPath(filepath.Join("fonts", "Inter-Bold.ttf"))
	italic := resolveAssetPath(filepath.Join("fonts", "Inter-Italic.ttf"))
	boldItalic := resolveAssetPath(filepath.Join("fonts", "Inter-BoldItalic.ttf"))

	if regular != "" && bold != "" {
		if italic == "" {
			italic = regular
		}
		if boldItalic == "" {
			boldItalic = bold
		}
		pdf.AddUTF8Font("Inter", "", regular)
		pdf.AddUTF8Font("Inter", "B", bold)
		pdf.AddUTF8Font("Inter", "I", italic)
		pdf.AddUTF8Font("Inter", "BI", boldItalic)
		if pdf.Ok() {
			return pdfFontConfig{family: "Inter", unicode: true}
		}
		pdf.ClearError()
	}

	return pdfFontConfig{family: "Helvetica", unicode: false}
}

type PreviewTemplatePDFRequest struct {
	Body string `json:"body" binding:"required"`
}

type GeneratePDFPersonalInput struct {
	FullName    string `json:"fullName"`
	PANNumber   string `json:"panNumber"`
	DOB         string `json:"dob"`
	Phone       string `json:"phone"`
	Email       string `json:"email"`
	FullAddress string `json:"fullAddress"`
}

type GeneratePDFDetailsInput struct {
	Duration       string     `json:"duration"`
	ConsultingFees float64    `json:"consultingFees"`
	StartDate      *time.Time `json:"startDate"`
	ExecutedDate   *time.Time `json:"executedDate"`
	MonthlyIncome  float64    `json:"monthlyIncome"`
}

type GenerateAgreementPDFRequest struct {
	LeadID    string                   `json:"leadId" binding:"required,max=24"`
	Personal  GeneratePDFPersonalInput `json:"personal"`
	Lenders   []leadmodels.Lender      `json:"lenders"`
	Agreement GeneratePDFDetailsInput  `json:"agreement"`
}

func cleanText(cfg pdfFontConfig, s string) string {
	if cfg.unicode {
		return s
	}
	return sanitizeForLatin1PDF(s)
}

// GeneratePDF renders the saved Agreement Template and structured Annexures A-D into an A4 PDF on the backend.
func GeneratePDF(c *gin.Context) {
	var rawReq GenerateAgreementPDFRequest
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	req := CreateAgreementRequest{
		LeadID:  rawReq.LeadID,
		Lenders: rawReq.Lenders,
		Personal: AgreementPersonalInput{
			FullName:    rawReq.Personal.FullName,
			PANNumber:   rawReq.Personal.PANNumber,
			DOB:         rawReq.Personal.DOB,
			Phone:       rawReq.Personal.Phone,
			Email:       rawReq.Personal.Email,
			FullAddress: rawReq.Personal.FullAddress,
		},
		Agreement: AgreementDetailsInput{
			Duration:       rawReq.Agreement.Duration,
			ConsultingFees: rawReq.Agreement.ConsultingFees,
			StartDate:      rawReq.Agreement.StartDate,
			ExecutedDate:   rawReq.Agreement.ExecutedDate,
			MonthlyIncome:  rawReq.Agreement.MonthlyIncome,
		},
	}

	leadObjID, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.LeadID))
	if err != nil || leadObjID.IsZero() {
		response.BadRequest(c, "Invalid lead ID format")
		return
	}

	ctx := c.Request.Context()

	tplCol := database.GetCollection("agreement_templates")
	if tplCol == nil {
		response.InternalServerError(c, "Database service unavailable")
		return
	}

	var tpl AgreementTemplate
	if err := tplCol.FindOne(ctx, bson.M{"key": defaultAgreementTemplateKey}).Decode(&tpl); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			response.BadRequest(c, "Please configure and save the Agreement Template first")
			return
		}
		log.Printf("[ERROR] [AgreementPDF] Template lookup failed: %v", err)
		response.InternalServerError(c, "Failed to load agreement template")
		return
	}
	if strings.TrimSpace(tpl.Body) == "" {
		response.BadRequest(c, "Please configure and save the Agreement Template first")
		return
	}

	leadCol := database.GetCollection("leads")
	var existing leadmodels.Lead
	if leadCol != nil {
		_ = leadCol.FindOne(ctx, bson.M{"_id": leadObjID, "deleted_at": bson.M{"$exists": false}}).Decode(&existing)
	}

	preparedBy := ""
	if existing.Agreement != nil && strings.TrimSpace(existing.Agreement.PreparedByName) != "" {
		preparedBy = existing.Agreement.PreparedByName
	} else if emp, ok := utils.GetAuthEmployee(c); ok && emp != nil {
		preparedBy = emp.Name
	}

	now := time.Now().Local()
	if req.Agreement.ExecutedDate == nil || req.Agreement.ExecutedDate.IsZero() {
		req.Agreement.ExecutedDate = &now
	}
	if req.Agreement.StartDate == nil || req.Agreement.StartDate.IsZero() {
		req.Agreement.StartDate = req.Agreement.ExecutedDate
	}

	buf, err := buildAgreementPDFBytes(tpl.Body, req, preparedBy)
	if err != nil {
		log.Printf("[ERROR] [AgreementPDF] PDF render failed: %v", err)
		response.InternalServerError(c, "Failed to generate PDF")
		return
	}

	filename := "Agreement.pdf"
	if existing.LeadNumber != "" {
		filename = fmt.Sprintf("Agreement_%s.pdf", existing.LeadNumber)
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/pdf", buf)
}

// PreviewTemplatePDF renders a draft or saved Agreement Template with representative sample data for in-modal preview.
func PreviewTemplatePDF(c *gin.Context) {
	var req PreviewTemplatePDFRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}
	if strings.TrimSpace(req.Body) == "" {
		response.BadRequest(c, "Agreement template cannot be empty")
		return
	}

	preparedBy := "SettleXpert Consultant"
	if emp, ok := utils.GetAuthEmployee(c); ok && emp != nil && strings.TrimSpace(emp.Name) != "" {
		preparedBy = emp.Name
	}

	now := time.Now().Local()
	sampleReq := CreateAgreementRequest{
		Personal: AgreementPersonalInput{
			FullName:    "SANCHITA CHATTERJEE",
			PANNumber:   "BTWPC6838E",
			DOB:         "1990-01-01",
			Phone:       "+91 70441 18629",
			Email:       "acsanchita@gmail.com",
			FullAddress: "C/O Arka Chatterjee, 17/A, Ram Ratan Bose Lane, Deshbondhu Park, Shyambazar, Kolkata, West Bengal - 700004",
		},
		Lenders: []leadmodels.Lender{
			{LenderName: "HDFC Bank", LoanType: "Personal Loan", OutstandingAmount: 150000},
			{LenderName: "ICICI Bank", LoanType: "Credit Card", OutstandingAmount: 80000},
		},
		Agreement: AgreementDetailsInput{
			Duration:       "6 Months",
			ConsultingFees: 7500,
			StartDate:      &now,
			ExecutedDate:   &now,
			MonthlyIncome:  45000,
		},
	}

	buf, err := buildAgreementPDFBytes(req.Body, sampleReq, preparedBy)
	if err != nil {
		log.Printf("[ERROR] [AgreementPDFPreview] PDF render failed: %v", err)
		response.InternalServerError(c, "Failed to generate PDF preview")
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `inline; filename="Agreement_Template_Preview.pdf"`)
	c.Data(http.StatusOK, "application/pdf", buf)
}

func buildAgreementPDFBytes(templateBody string, req CreateAgreementRequest, preparedBy string) ([]byte, error) {
	mainBody := extractMainAgreementBody(templateBody)
	replacements := buildPlaceholderReplacements(req, preparedBy)
	logoPath := resolveLogoPath()

	pdf := fpdf.New("P", "mm", "A4", "")
	fontCfg := configurePDFFonts(pdf)
	pdf.SetMargins(pageMarginL, pageTopY, pageMarginR)
	pdf.SetAutoPageBreak(true, 18)
	pdf.AliasNbPages("")

	pdf.SetHeaderFunc(func() {
		renderPDFHeader(pdf, fontCfg, logoPath)
	})
	pdf.SetFooterFunc(func() {
		renderPDFFooter(pdf, fontCfg)
	})

	// Main Consultancy Agreement
	pdf.AddPage()
	renderMainAgreementLines(pdf, fontCfg, mainBody, replacements)
	renderSignatureBlock(pdf, fontCfg, req.Personal.FullName)

	// Annexure A
	pdf.AddPage()
	renderAnnexureA(pdf, fontCfg, req)

	// Annexure B
	pdf.AddPage()
	renderAnnexureB(pdf, fontCfg, req, preparedBy)

	// Annexure C & D
	pdf.AddPage()
	renderAnnexureCD(pdf, fontCfg, req)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func renderPDFHeader(pdf *fpdf.Fpdf, fontCfg pdfFontConfig, logoPath string) {
	if logoPath != "" {
		pdf.ImageOptions(logoPath, pageMarginL, 10.0, 35, 0, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	} else {
		pdf.SetXY(pageMarginL, 11.5)
		pdf.SetFont(fontCfg.family, "B", 14)
		pdf.SetTextColor(colorGreen700.r, colorGreen700.g, colorGreen700.b)
		pdf.CellFormat(42, 6, "SettleXpert", "", 0, "L", false, 0, "")
	}

	rightX := 210.0 - pageMarginR

	pdf.SetXY(72, 9.5)
	pdf.SetFont(fontCfg.family, "B", 10.5)
	pdf.SetTextColor(colorSlate900.r, colorSlate900.g, colorSlate900.b)
	pdf.CellFormat(rightX-72, 4.6, "SETTLEXPERT LLP", "", 1, "R", false, 0, "")

	pdf.SetXY(72, 14.3)
	pdf.SetFont(fontCfg.family, "", 7.8)
	pdf.SetTextColor(colorSlate800.r, colorSlate800.g, colorSlate800.b)
	pdf.CellFormat(rightX-72, 3.8, "CB-201, Naraina Vihar, Ring Road, New Delhi, Delhi, India", "", 1, "R", false, 0, "")

	bullet := cleanText(fontCfg, " • Email: ")
	line3 := []textSegment{
		{text: "Phone: ", color: &colorSlate600},
		{text: "+91 89292 23949", bold: true, color: &colorSlate900},
		{text: bullet, color: &colorSlate600},
		{text: "info@settlexpert.com", bold: true, color: &colorSlate900},
	}
	renderRightAlignedSegments(pdf, fontCfg, rightX, 18.2, 3.8, 7.8, line3)

	line4 := []textSegment{
		{text: "Website: ", color: &colorSlate600},
		{text: "www.settlexpert.com", bold: true, color: &colorSlate900},
	}
	renderRightAlignedSegments(pdf, fontCfg, rightX, 22.0, 3.8, 7.8, line4)

	pdf.SetDrawColor(colorGreen700.r, colorGreen700.g, colorGreen700.b)
	pdf.SetLineWidth(0.55)
	pdf.Line(pageMarginL, 27.5, rightX, 27.5)
	pdf.SetXY(pageMarginL, pageTopY)
}

func renderRightAlignedSegments(pdf *fpdf.Fpdf, fontCfg pdfFontConfig, rightX, y, h, size float64, segs []textSegment) {
	prevMargin := pdf.GetCellMargin()
	pdf.SetCellMargin(0)
	defer pdf.SetCellMargin(prevMargin)

	totalW := 0.0
	for _, s := range segs {
		style := ""
		if s.bold {
			style = "B"
		}
		pdf.SetFont(fontCfg.family, style, size)
		totalW += pdf.GetStringWidth(s.text)
	}
	curX := rightX - totalW
	pdf.SetXY(curX, y)
	for _, s := range segs {
		style := ""
		if s.bold {
			style = "B"
		}
		c := colorSlate800
		if s.color != nil {
			c = *s.color
		}
		pdf.SetFont(fontCfg.family, style, size)
		pdf.SetTextColor(c.r, c.g, c.b)
		w := pdf.GetStringWidth(s.text)
		pdf.CellFormat(w, h, s.text, "", 0, "L", false, 0, "")
	}
}

func renderPDFFooter(pdf *fpdf.Fpdf, fontCfg pdfFontConfig) {
	pdf.SetY(-15)
	pdf.SetDrawColor(colorBorder.r, colorBorder.g, colorBorder.b)
	pdf.SetLineWidth(0.25)
	pdf.Line(pageMarginL, 281, 210.0-pageMarginR, 281)
	pdf.SetY(282.5)
	pdf.SetFont(fontCfg.family, "", 7.8)
	pdf.SetTextColor(colorSlate500.r, colorSlate500.g, colorSlate500.b)
	footerText := cleanText(fontCfg, fmt.Sprintf("Email: info@settlexpert.com | Phone: +91 89292 23949 | Website: www.settlexpert.com  •  Page %d of {nb}", pdf.PageNo()))
	pdf.CellFormat(0, 5.5, footerText, "", 0, "C", false, 0, "")
}

func extractMainAgreementBody(template string) string {
	normalized := strings.ReplaceAll(template, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")

	// Skip duplicate leading letterhead lines above CONSULTANCY AGREEMENT if present near top.
	startIdx := 0
	for i := 0; i < len(lines) && i < 25; i++ {
		if strings.EqualFold(strings.TrimSpace(lines[i]), "CONSULTANCY AGREEMENT") {
			startIdx = i
			break
		}
	}

	endIdx := len(lines)
	for i := startIdx; i < len(lines); i++ {
		upper := strings.ToUpper(strings.TrimSpace(lines[i]))
		if upper == "" {
			continue
		}
		if strings.HasPrefix(upper, "ANNEXURE A") ||
			strings.HasPrefix(upper, "ANNEXURE - A") ||
			upper == "FIRST PARTY" ||
			strings.HasPrefix(upper, "FIRST PARTY ") ||
			strings.HasPrefix(upper, "FIRST PARTY\t") {
			endIdx = i
			break
		}
	}

	// Trim trailing separator lines (==== or blank lines) right before signature/Annexure A.
	for endIdx > startIdx {
		prev := strings.TrimSpace(lines[endIdx-1])
		if prev == "" || strings.HasPrefix(prev, "====") || strings.HasPrefix(prev, "----") {
			endIdx--
			continue
		}
		break
	}

	return strings.TrimSpace(strings.Join(lines[startIdx:endIdx], "\n"))
}

func renderMainAgreementLines(pdf *fpdf.Fpdf, fontCfg pdfFontConfig, body string, replacements map[string]string) {
	lines := strings.Split(body, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			pdf.Ln(2.5)
			continue
		}
		if strings.HasPrefix(trimmed, "====") {
			continue
		}

		switch {
		case strings.EqualFold(trimmed, "CONSULTANCY AGREEMENT"):
			renderCenteredUnderlinedTitle(pdf, fontCfg, "CONSULTANCY AGREEMENT", 13)
			pdf.Ln(3)

		case numberedSectionRe.MatchString(trimmed):
			if pdf.GetY() > 252 {
				pdf.AddPage()
			}
			pdf.Ln(2.5)
			pdf.SetX(pageMarginL)
			pdf.SetFont(fontCfg.family, "B", 10.8)
			pdf.SetTextColor(colorSlate900.r, colorSlate900.g, colorSlate900.b)
			pdf.MultiCell(pageContentW, 5.6, cleanText(fontCfg, trimmed), "", "L", false)
			pdf.Ln(1.2)

		case strings.HasPrefix(trimmed, "NOW THIS AGREEMENT WITNESSETH"):
			if pdf.GetY() > 260 {
				pdf.AddPage()
			}
			pdf.Ln(2.5)
			pdf.SetX(pageMarginL)
			pdf.SetFont(fontCfg.family, "B", 10.2)
			pdf.SetTextColor(colorSlate900.r, colorSlate900.g, colorSlate900.b)
			pdf.MultiCell(pageContentW, 5.5, cleanText(fontCfg, trimmed), "", "C", false)
			pdf.Ln(2)

		case strings.EqualFold(trimmed, "AND"):
			pdf.Ln(1)
			pdf.SetX(pageMarginL)
			pdf.SetFont(fontCfg.family, "B", 10.2)
			pdf.SetTextColor(colorSlate900.r, colorSlate900.g, colorSlate900.b)
			pdf.MultiCell(pageContentW, 5.4, "AND", "", "L", false)
			pdf.Ln(1)

		case strings.HasPrefix(trimmed, "- "):
			bulletChar := cleanText(fontCfg, "•")
			itemText := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			if pdf.GetY() > 270 {
				pdf.AddPage()
			}
			y := pdf.GetY()
			pdf.SetFont(fontCfg.family, "B", 10)
			pdf.SetTextColor(colorSlate900.r, colorSlate900.g, colorSlate900.b)
			pdf.SetXY(pageMarginL+6, y)
			pdf.CellFormat(4.5, 5.4, bulletChar, "", 0, "L", false, 0, "")
			segs := buildRichParagraphSegments(fontCfg, itemText, replacements)
			renderRichWrappedText(pdf, fontCfg, pageMarginL+10.5, pageContentW-10.5, 5.4, 10, segs)
			pdf.Ln(1.5)

		case subItemRe.MatchString(trimmed):
			m := subItemRe.FindStringSubmatch(trimmed)
			segs := []textSegment{
				{text: m[1], bold: true, color: &colorSlate900},
			}
			segs = append(segs, buildRichParagraphSegments(fontCfg, m[2], replacements)...)
			renderRichWrappedText(pdf, fontCfg, pageMarginL+5, pageContentW-5, 5.4, 10, segs)
			pdf.Ln(1.5)

		case strings.HasPrefix(trimmed, "Grievance & Escalation Email:"):
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "Grievance & Escalation Email:"))
			segs := []textSegment{
				{text: "Grievance & Escalation Email:", bold: true, color: &colorSlate900},
			}
			if rest != "" {
				renderRichWrappedText(pdf, fontCfg, pageMarginL, pageContentW, 5.4, 10, segs)
				pdf.Ln(0.8)
				emailSeg := []textSegment{{text: cleanText(fontCfg, rest), bold: true, color: &colorGreen700}}
				renderRichWrappedText(pdf, fontCfg, pageMarginL, pageContentW, 5.4, 10, emailSeg)
			} else {
				renderRichWrappedText(pdf, fontCfg, pageMarginL, pageContentW, 5.4, 10, segs)
			}
			pdf.Ln(1.5)

		case strings.EqualFold(trimmed, "info@settlexpert.com"):
			renderRichWrappedText(pdf, fontCfg, pageMarginL, pageContentW, 5.4, 10, []textSegment{
				{text: "info@settlexpert.com", bold: true, color: &colorGreen700},
			})
			pdf.Ln(1.5)

		default:
			segs := buildRichParagraphSegments(fontCfg, trimmed, replacements)
			renderRichWrappedText(pdf, fontCfg, pageMarginL, pageContentW, 5.4, 10, segs)
			pdf.Ln(1.8)
		}
	}
}

func buildRichParagraphSegments(cfg pdfFontConfig, line string, replacements map[string]string) []textSegment {
	// First substitute {{placeholders}} while recording their values as bold segments
	var segs []textSegment
	cur := line
	for len(cur) > 0 {
		earliestIdx := -1
		earliestToken := ""
		for token := range replacements {
			if idx := strings.Index(cur, token); idx != -1 && (earliestIdx == -1 || idx < earliestIdx) {
				earliestIdx = idx
				earliestToken = token
			}
		}
		if earliestIdx == -1 {
			segs = append(segs, splitKeyLegalBold(cfg, cur)...)
			break
		}
		if earliestIdx > 0 {
			segs = append(segs, splitKeyLegalBold(cfg, cur[:earliestIdx])...)
		}
		val := cleanText(cfg, replacements[earliestToken])
		segs = append(segs, textSegment{text: val, bold: true, color: &colorSlate900})
		cur = cur[earliestIdx+len(earliestToken):]
	}

	return segs
}

func splitKeyLegalBold(cfg pdfFontConfig, chunk string) []textSegment {
	boldPhrases := []string{
		"M/s. SettleXpert LLP",
		"CB-201, Naraina Vihar, Ring Road, New Delhi, Delhi, India",
		`"First Party"`,
		`"Company"`,
		`"Second Party"`,
		`"Client"`,
		`"Parties"`,
		`"Party"`,
		"NOW, THEREFORE,",
		"WHEREAS,",
		"Annexure A",
		"Annexure B & C",
		"Annexure B",
	}

	// Sort longest first so longer matches take precedence
	sort.SliceStable(boldPhrases, func(i, j int) bool {
		return len(boldPhrases[i]) > len(boldPhrases[j])
	})

	var out []textSegment
	remaining := chunk
	for len(remaining) > 0 {
		earliestIdx := -1
		matched := ""
		for _, phrase := range boldPhrases {
			if idx := strings.Index(remaining, phrase); idx != -1 && (earliestIdx == -1 || idx < earliestIdx) {
				earliestIdx = idx
				matched = phrase
			}
		}
		if earliestIdx == -1 {
			out = append(out, textSegment{text: cleanText(cfg, remaining), color: &colorSlate800})
			break
		}
		if earliestIdx > 0 {
			out = append(out, textSegment{text: cleanText(cfg, remaining[:earliestIdx]), color: &colorSlate800})
		}
		out = append(out, textSegment{text: cleanText(cfg, matched), bold: true, color: &colorSlate900})
		remaining = remaining[earliestIdx+len(matched):]
	}
	return out
}

// renderRichWrappedText wraps mixed Regular/Bold segments across lines cleanly without overlapping or mid-word breaks.
func renderRichWrappedText(pdf *fpdf.Fpdf, fontCfg pdfFontConfig, x, maxW, lineH, fontSize float64, segs []textSegment) {
	type wordToken struct {
		text  string
		bold  bool
		color rgbColor
		space bool
	}

	var tokens []wordToken
	for _, seg := range segs {
		c := colorSlate800
		if seg.color != nil {
			c = *seg.color
		}
		// Split into alternating non-space words and single-space separators
		i := 0
		for i < len(seg.text) {
			if seg.text[i] == ' ' || seg.text[i] == '\t' {
				j := i
				for j < len(seg.text) && (seg.text[j] == ' ' || seg.text[j] == '\t') {
					j++
				}
				tokens = append(tokens, wordToken{text: " ", bold: seg.bold, color: c, space: true})
				i = j
				continue
			}
			j := i
			for j < len(seg.text) && seg.text[j] != ' ' && seg.text[j] != '\t' {
				j++
			}
			tokens = append(tokens, wordToken{text: seg.text[i:j], bold: seg.bold, color: c})
			i = j
		}
	}

	measureToken := func(t wordToken) float64 {
		style := ""
		if t.bold {
			style = "B"
		}
		pdf.SetFont(fontCfg.family, style, fontSize)
		return pdf.GetStringWidth(t.text)
	}

	drawLine := func(line []wordToken) {
		// Trim trailing space
		for len(line) > 0 && line[len(line)-1].space {
			line = line[:len(line)-1]
		}
		if pdf.GetY()+lineH > 276 {
			pdf.AddPage()
		}
		prevMargin := pdf.GetCellMargin()
		pdf.SetCellMargin(0)
		pdf.SetX(x)
		for _, t := range line {
			style := ""
			if t.bold {
				style = "B"
			}
			pdf.SetFont(fontCfg.family, style, fontSize)
			pdf.SetTextColor(t.color.r, t.color.g, t.color.b)
			w := pdf.GetStringWidth(t.text)
			pdf.CellFormat(w, lineH, t.text, "", 0, "L", false, 0, "")
		}
		pdf.SetCellMargin(prevMargin)
		pdf.Ln(lineH)
	}

	var curLine []wordToken
	curW := 0.0

	for _, tok := range tokens {
		if tok.space && len(curLine) == 0 {
			continue
		}
		tw := measureToken(tok)
		if !tok.space && len(curLine) > 0 && curW+tw > maxW {
			drawLine(curLine)
			curLine = nil
			curW = 0
		}
		if tok.space && len(curLine) == 0 {
			continue
		}
		curLine = append(curLine, tok)
		curW += tw
	}
	if len(curLine) > 0 {
		drawLine(curLine)
	}
}

func renderSignatureBlock(pdf *fpdf.Fpdf, fontCfg pdfFontConfig, clientName string) {
	if pdf.GetY() > 240 {
		pdf.AddPage()
	}
	pdf.Ln(8)
	y := pdf.GetY()

	col2X := pageMarginL + 94

	pdf.SetFont(fontCfg.family, "B", 10.2)
	pdf.SetTextColor(colorSlate900.r, colorSlate900.g, colorSlate900.b)
	pdf.SetXY(pageMarginL, y)
	pdf.CellFormat(84, 5.5, "FIRST PARTY", "", 0, "L", false, 0, "")
	pdf.SetXY(col2X, y)
	pdf.CellFormat(80, 5.5, "SECOND PARTY", "", 1, "L", false, 0, "")

	pdf.Ln(1.5)
	prevMargin := pdf.GetCellMargin()
	pdf.SetCellMargin(0)
	y = pdf.GetY()
	pdf.SetXY(pageMarginL, y)
	pdf.SetFont(fontCfg.family, "B", 10)
	pdf.CellFormat(pdf.GetStringWidth("NAME: "), 5.5, "NAME: ", "", 0, "L", false, 0, "")
	pdf.SetFont(fontCfg.family, "", 10)
	pdf.CellFormat(70, 5.5, "SettleXpert LLP", "", 0, "L", false, 0, "")

	pdf.SetXY(col2X, y)
	pdf.SetFont(fontCfg.family, "B", 10)
	pdf.CellFormat(pdf.GetStringWidth("NAME: "), 5.5, "NAME: ", "", 0, "L", false, 0, "")
	pdf.SetFont(fontCfg.family, "", 10)
	pdf.CellFormat(68, 5.5, valueOrDash(cleanText(fontCfg, clientName)), "", 1, "L", false, 0, "")
	pdf.SetCellMargin(prevMargin)

	pdf.Ln(5)
	y = pdf.GetY()
	pdf.SetFont(fontCfg.family, "B", 10)
	pdf.SetXY(pageMarginL, y)
	pdf.CellFormat(84, 5.5, "Signature: ______________________", "", 0, "L", false, 0, "")
	pdf.SetXY(col2X, y)
	pdf.CellFormat(80, 5.5, "Signature: ______________________", "", 1, "L", false, 0, "")
}

func renderAnnexureA(pdf *fpdf.Fpdf, fontCfg pdfFontConfig, req CreateAgreementRequest) {
	renderCenteredUnderlinedTitle(pdf, fontCfg, "Annexure A - Client & Lender Information", 12.5)
	pdf.Ln(6)

	// Table 1: Client Information
	col1W := 56.0
	col2W := pageContentW - col1W

	pdf.SetFillColor(colorGreenHdr.r, colorGreenHdr.g, colorGreenHdr.b)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetDrawColor(colorBorder.r, colorBorder.g, colorBorder.b)
	pdf.SetLineWidth(0.25)
	pdf.SetFont(fontCfg.family, "B", 9.5)
	pdf.SetX(pageMarginL)
	pdf.CellFormat(col1W, 8.6, "   Items", "1", 0, "L", true, 0, "")
	pdf.CellFormat(col2W, 8.6, "   Details", "1", 1, "L", true, 0, "")

	clientRows := []struct {
		label string
		val   string
		bold  bool
	}{
		{"Name", valueOrDash(cleanText(fontCfg, req.Personal.FullName)), true},
		{"PAN Number", valueOrDash(strings.ToUpper(req.Personal.PANNumber)), true},
		{"Date of Birth", formatPDFDate(req.Personal.DOB), false},
		{"Phone Number", valueOrDash(req.Personal.Phone), false},
		{"Email", valueOrDash(req.Personal.Email), false},
		{"Start Date", formatTimePtrPDF(req.Agreement.StartDate), false},
	}

	for _, r := range clientRows {
		pdf.SetX(pageMarginL)
		pdf.SetTextColor(colorSlate800.r, colorSlate800.g, colorSlate800.b)
		pdf.SetFont(fontCfg.family, "", 9.5)
		pdf.CellFormat(col1W, 8.2, "   "+r.label, "1", 0, "L", false, 0, "")

		style := ""
		if r.bold {
			style = "B"
			pdf.SetTextColor(colorSlate900.r, colorSlate900.g, colorSlate900.b)
		}
		pdf.SetFont(fontCfg.family, style, 9.5)
		pdf.CellFormat(col2W, 8.2, "   "+r.val, "1", 1, "L", false, 0, "")
	}

	pdf.Ln(6)

	// Table 2: Lenders Information
	lCol1 := 64.0
	lCol2 := 60.0
	lCol3 := pageContentW - lCol1 - lCol2

	pdf.SetFillColor(colorGreenHdr.r, colorGreenHdr.g, colorGreenHdr.b)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont(fontCfg.family, "B", 9.5)
	pdf.SetX(pageMarginL)
	pdf.CellFormat(lCol1, 8.6, "   Lender Name*", "1", 0, "L", true, 0, "")
	pdf.CellFormat(lCol2, 8.6, "   Loan Type (PL/CC)", "1", 0, "L", true, 0, "")
	pdf.CellFormat(lCol3, 8.6, "Loan Amt (Balance)   ", "1", 1, "R", true, 0, "")

	var totalOutstanding float64
	if len(req.Lenders) == 0 {
		pdf.SetX(pageMarginL)
		pdf.SetTextColor(colorSlate500.r, colorSlate500.g, colorSlate500.b)
		pdf.SetFont(fontCfg.family, "I", 9.5)
		pdf.CellFormat(pageContentW, 8.6, "No lenders selected", "1", 1, "C", false, 0, "")
	} else {
		for _, l := range req.Lenders {
			pdf.SetX(pageMarginL)
			pdf.SetTextColor(colorSlate800.r, colorSlate800.g, colorSlate800.b)
			pdf.SetFont(fontCfg.family, "", 9.5)
			pdf.CellFormat(lCol1, 8.2, "   "+valueOrDash(cleanText(fontCfg, l.LenderName)), "1", 0, "L", false, 0, "")
			pdf.CellFormat(lCol2, 8.2, "   "+valueOrDash(cleanText(fontCfg, l.LoanType)), "1", 0, "L", false, 0, "")
			pdf.SetTextColor(colorSlate900.r, colorSlate900.g, colorSlate900.b)
			pdf.SetFont(fontCfg.family, "B", 9.5)
			pdf.CellFormat(lCol3, 8.2, formatIndianNumber(l.OutstandingAmount)+"   ", "1", 1, "R", false, 0, "")
			totalOutstanding += l.OutstandingAmount
		}
	}

	pdf.SetX(pageMarginL)
	pdf.SetFillColor(colorRowTint.r, colorRowTint.g, colorRowTint.b)
	pdf.SetTextColor(colorSlate900.r, colorSlate900.g, colorSlate900.b)
	pdf.SetFont(fontCfg.family, "B", 9.5)
	pdf.CellFormat(lCol1+lCol2, 8.6, "Total", "1", 0, "C", true, 0, "")
	pdf.SetTextColor(colorGreen700.r, colorGreen700.g, colorGreen700.b)
	pdf.CellFormat(lCol3, 8.6, formatIndianNumber(totalOutstanding)+"   ", "1", 1, "R", true, 0, "")

	pdf.Ln(3)
	pdf.SetX(pageMarginL)
	pdf.SetFont(fontCfg.family, "I", 8.5)
	pdf.SetTextColor(colorSlate500.r, colorSlate500.g, colorSlate500.b)
	pdf.MultiCell(pageContentW, 4.5, "* Each lender has a unique process and resolution is at the discretion of the lenders", "", "L", false)
}

func renderAnnexureB(pdf *fpdf.Fpdf, fontCfg pdfFontConfig, req CreateAgreementRequest, preparedBy string) {
	renderCenteredUnderlinedTitle(pdf, fontCfg, "Annexure B", 12.5)
	pdf.Ln(6)

	var totalLoan float64
	for _, l := range req.Lenders {
		totalLoan += l.OutstandingAmount
	}

	durationDisplay := formatDurationDisplay(req.Agreement.Duration)

	drawPair := func(l1, v1, l2, v2 string) {
		prevMargin := pdf.GetCellMargin()
		pdf.SetCellMargin(0)
		y := pdf.GetY()
		pdf.SetXY(pageMarginL, y)
		pdf.SetTextColor(colorSlate900.r, colorSlate900.g, colorSlate900.b)
		pdf.SetFont(fontCfg.family, "B", 9.5)
		lw1 := pdf.GetStringWidth(l1) + 2.5
		pdf.CellFormat(lw1, 6.2, l1, "", 0, "L", false, 0, "")
		pdf.SetFont(fontCfg.family, "B", 9.5)
		pdf.CellFormat(90-lw1, 6.2, v1, "", 0, "L", false, 0, "")

		if l2 != "" {
			pdf.SetXY(pageMarginL+92, y)
			pdf.SetFont(fontCfg.family, "B", 9.5)
			lw2 := pdf.GetStringWidth(l2) + 2.5
			pdf.CellFormat(lw2, 6.2, l2, "", 0, "L", false, 0, "")
			pdf.CellFormat(82-lw2, 6.2, v2, "", 0, "L", false, 0, "")
		}
		pdf.SetCellMargin(prevMargin)
		pdf.Ln(7.5)
	}

	drawPair("Client Name:", valueOrDash(cleanText(fontCfg, req.Personal.FullName)), "Prepared By:", valueOrDash(cleanText(fontCfg, preparedBy)))
	drawPair("Total Loan:", "Rs. "+formatIndianNumber(totalLoan), "Monthly Income:", "Rs. "+formatIndianNumber(req.Agreement.MonthlyIncome))
	drawPair("Agreement Duration:", durationDisplay, "", "")
	drawPair("Consultancy Fees:", "Rs. "+formatIndianNumber(req.Agreement.ConsultingFees), "Resolution Duration:", durationDisplay)

	pdf.Ln(2)
	pdf.SetX(pageMarginL)
	pdf.SetFont(fontCfg.family, "", 8.5)
	pdf.SetTextColor(colorSlate500.r, colorSlate500.g, colorSlate500.b)
	pdf.MultiCell(pageContentW, 4.5, "*This is based on past precedents. Each case is unique and at the discretion of the lenders.", "", "L", false)
	pdf.SetX(pageMarginL)
	pdf.MultiCell(pageContentW, 4.5, "**To get faster resolutions, save as much as possible on monthly basis.", "", "L", false)

	pdf.Ln(5.5)
	pdf.SetX(pageMarginL)
	pdf.SetFont(fontCfg.family, "B", 10.5)
	pdf.SetTextColor(colorSlate900.r, colorSlate900.g, colorSlate900.b)
	pdf.CellFormat(pageContentW, 6, "Consultancy Fee Schedule", "", 1, "L", false, 0, "")
	pdf.Ln(2.5)

	months := 1
	if m := durationDigitsRe.FindStringSubmatch(req.Agreement.Duration); len(m) > 1 {
		if parsed, err := strconv.Atoi(m[1]); err == nil && parsed > 0 && parsed <= 60 {
			months = parsed
		}
	}

	baseDate := time.Now().Local()
	if req.Agreement.StartDate != nil && !req.Agreement.StartDate.IsZero() {
		baseDate = req.Agreement.StartDate.Local()
	}

	c1, c2, c3 := 44.0, 68.0, pageContentW-112.0
	drawScheduleHeader := func() {
		pdf.SetX(pageMarginL)
		pdf.SetFillColor(colorSlateHdr.r, colorSlateHdr.g, colorSlateHdr.b)
		pdf.SetDrawColor(colorBorder.r, colorBorder.g, colorBorder.b)
		pdf.SetLineWidth(0.25)
		pdf.SetTextColor(colorSlate900.r, colorSlate900.g, colorSlate900.b)
		pdf.SetFont(fontCfg.family, "B", 9.5)
		pdf.CellFormat(c1, 8.8, "Month", "1", 0, "C", true, 0, "")
		pdf.CellFormat(c2, 8.8, "Consultancy Fees", "1", 0, "C", true, 0, "")
		pdf.CellFormat(c3, 8.8, "Due Date", "1", 1, "C", true, 0, "")
	}

	drawScheduleHeader()

	feeStr := "Rs. " + formatIndianNumber(req.Agreement.ConsultingFees)
	for i := 0; i < months; i++ {
		if pdf.GetY()+8.4 > 276 {
			pdf.AddPage()
			drawScheduleHeader()
		}
		due := baseDate.AddDate(0, i, 0).Format("02 Jan 2006")
		pdf.SetX(pageMarginL)
		pdf.SetDrawColor(colorBorder.r, colorBorder.g, colorBorder.b)
		pdf.SetTextColor(colorSlate800.r, colorSlate800.g, colorSlate800.b)
		pdf.SetFont(fontCfg.family, "", 9.5)
		pdf.CellFormat(c1, 8.2, strconv.Itoa(i+1), "1", 0, "C", false, 0, "")
		pdf.SetTextColor(colorSlate900.r, colorSlate900.g, colorSlate900.b)
		pdf.SetFont(fontCfg.family, "B", 9.5)
		pdf.CellFormat(c2, 8.2, feeStr, "1", 0, "C", false, 0, "")
		pdf.SetTextColor(colorSlate800.r, colorSlate800.g, colorSlate800.b)
		pdf.SetFont(fontCfg.family, "", 9.5)
		pdf.CellFormat(c3, 8.2, due, "1", 1, "C", false, 0, "")
	}
}

func renderAnnexureCD(pdf *fpdf.Fpdf, fontCfg pdfFontConfig, req CreateAgreementRequest) {
	clientName := valueOrDash(cleanText(fontCfg, req.Personal.FullName))
	execDate := formatTimePtrPDF(req.Agreement.ExecutedDate)
	incomeStr := "INR " + formatIndianNumber(req.Agreement.MonthlyIncome) + " /-"
	address := valueOrDash(cleanText(fontCfg, req.Personal.FullAddress))

	renderCenteredUnderlinedTitle(pdf, fontCfg, "Annexure C - Confirmation of in-hand monthly income", 12)
	pdf.Ln(5.5)

	p1Segs := []textSegment{
		{text: "I, ", color: &colorSlate800},
		{text: clientName, bold: true, color: &colorSlate900},
		{text: " expressly confirm that my monthly in hand income as of ", color: &colorSlate800},
		{text: execDate, bold: true, color: &colorSlate900},
		{text: " is ", color: &colorSlate800},
		{text: incomeStr, bold: true, color: &colorSlate900},
		{text: " per month after deductions.", color: &colorSlate800},
	}
	renderRichWrappedText(pdf, fontCfg, pageMarginL, pageContentW, 5.5, 10, p1Segs)
	pdf.Ln(3)

	renderRichWrappedText(pdf, fontCfg, pageMarginL, pageContentW, 5.5, 10, []textSegment{
		{text: "I understand that resolution with lenders is a complex process and income is a significant factor. I understand that if I provide factually incorrect income, the same can make my case weaker.", color: &colorSlate800},
	})
	pdf.Ln(3)

	renderRichWrappedText(pdf, fontCfg, pageMarginL, pageContentW, 5.5, 10, []textSegment{
		{text: "In case I have provided incorrect income information the First Party is entitled to:", color: &colorSlate800},
	})
	pdf.Ln(1.5)

	bullet := cleanText(fontCfg, "•")
	bullets := []string{
		"Seek past income proof in coming months based on attested bank statements.",
		"Modify the financial proposal suitably in case income is revised.",
	}
	for _, b := range bullets {
		y := pdf.GetY()
		pdf.SetXY(pageMarginL+5, y)
		pdf.SetFont(fontCfg.family, "B", 10)
		pdf.SetTextColor(colorSlate900.r, colorSlate900.g, colorSlate900.b)
		pdf.CellFormat(4.5, 5.5, bullet, "", 0, "L", false, 0, "")
		renderRichWrappedText(pdf, fontCfg, pageMarginL+9.5, pageContentW-9.5, 5.5, 10, []textSegment{
			{text: b, color: &colorSlate800},
		})
		pdf.Ln(1)
	}

	pdf.Ln(5.5)
	y := pdf.GetY()
	pdf.SetDrawColor(colorBorder.r, colorBorder.g, colorBorder.b)
	pdf.SetLineWidth(0.3)
	pdf.Line(pageMarginL, y, 210.0-pageMarginR, y)
	pdf.Ln(6)

	renderCenteredUnderlinedTitle(pdf, fontCfg, "Annexure D - Permission Letter to Communicate with Lenders and Collection Agencies", 12)
	pdf.Ln(5.5)

	p2Segs := []textSegment{
		{text: "I, ", color: &colorSlate800},
		{text: clientName, bold: true, color: &colorSlate900},
		{text: " expressly authorize ", color: &colorSlate800},
		{text: "SettleXpert LLP (Company)", bold: true, color: &colorSlate900},
		{text: ", its agents and representatives to communicate and negotiate with lenders and their collection agencies and to settle the loan amounts on my behalf in discussions with my lenders.", color: &colorSlate800},
	}
	renderRichWrappedText(pdf, fontCfg, pageMarginL, pageContentW, 5.5, 10, p2Segs)
	pdf.Ln(3.5)

	pdf.SetX(pageMarginL)
	pdf.SetFont(fontCfg.family, "B", 10)
	pdf.SetTextColor(colorSlate900.r, colorSlate900.g, colorSlate900.b)
	pdf.CellFormat(pageContentW, 5.5, "The Company is entitled to", "", 1, "L", false, 0, "")
	pdf.Ln(1.5)

	renderRichWrappedText(pdf, fontCfg, pageMarginL, pageContentW, 5.5, 10, []textSegment{
		{text: "Obtain on my behalf, records, debt validations and support for the debts allegedly owed by me. Communicate and negotiate with banks, lenders, financial institutions, licensed collection agencies and all other related entities and individuals relating to my debts.", color: &colorSlate800},
	})

	pdf.Ln(6)
	pdf.SetX(pageMarginL)
	pdf.SetFont(fontCfg.family, "B", 10)
	pdf.SetTextColor(colorSlate900.r, colorSlate900.g, colorSlate900.b)
	pdf.CellFormat(pageContentW, 5.5, "Signature: _________________________", "", 1, "L", false, 0, "")
	pdf.Ln(2)

	prevMargin := pdf.GetCellMargin()
	pdf.SetCellMargin(0)
	pdf.SetX(pageMarginL)
	pdf.SetFont(fontCfg.family, "B", 10)
	pdf.CellFormat(pdf.GetStringWidth("Date: "), 5.5, "Date: ", "", 0, "L", false, 0, "")
	pdf.SetFont(fontCfg.family, "", 10)
	pdf.CellFormat(60, 5.5, execDate, "", 1, "L", false, 0, "")
	pdf.Ln(2.5)

	placeY := pdf.GetY()
	pdf.SetXY(pageMarginL, placeY)
	pdf.SetFont(fontCfg.family, "B", 9.5)
	pdf.CellFormat(pdf.GetStringWidth("Place: "), 5.2, "Place: ", "", 0, "L", false, 0, "")
	pdf.SetFont(fontCfg.family, "", 9.5)
	pdf.SetTextColor(colorSlate800.r, colorSlate800.g, colorSlate800.b)
	pdf.MultiCell(110, 5.2, address, "", "L", false)

	pdf.SetXY(150, placeY)
	pdf.SetFont(fontCfg.family, "B", 9.5)
	pdf.SetTextColor(colorSlate900.r, colorSlate900.g, colorSlate900.b)
	pdf.CellFormat(42, 5.2, "Executed Date:", "", 1, "L", false, 0, "")
	pdf.SetX(150)
	pdf.SetFont(fontCfg.family, "", 9.5)
	pdf.SetTextColor(colorSlate800.r, colorSlate800.g, colorSlate800.b)
	pdf.CellFormat(42, 5.2, execDate, "", 1, "L", false, 0, "")
	pdf.SetCellMargin(prevMargin)
}

func renderCenteredUnderlinedTitle(pdf *fpdf.Fpdf, fontCfg pdfFontConfig, title string, size float64) {
	pdf.SetX(pageMarginL)
	pdf.SetFont(fontCfg.family, "BU", size)
	pdf.SetTextColor(colorSlate900.r, colorSlate900.g, colorSlate900.b)
	pdf.MultiCell(pageContentW, 6.4, cleanText(fontCfg, title), "", "C", false)
}

func buildPlaceholderReplacements(req CreateAgreementRequest, preparedBy string) map[string]string {
	var lendersLines []string
	var totalOutstanding float64
	for i, l := range req.Lenders {
		name := strings.TrimSpace(l.LenderName)
		if name == "" {
			continue
		}
		loanType := strings.TrimSpace(l.LoanType)
		if loanType == "" {
			loanType = "Loan"
		}
		lendersLines = append(lendersLines, fmt.Sprintf("%d. %s - %s - Rs. %s", i+1, name, loanType, formatIndianNumber(l.OutstandingAmount)))
		totalOutstanding += l.OutstandingAmount
	}

	lendersList := "None"
	if len(lendersLines) > 0 {
		lendersList = strings.Join(lendersLines, ", ")
	}

	months := 1
	if m := durationDigitsRe.FindStringSubmatch(req.Agreement.Duration); len(m) > 1 {
		if parsed, err := strconv.Atoi(m[1]); err == nil && parsed > 0 {
			months = parsed
		}
	}
	totalFee := req.Agreement.ConsultingFees * float64(months)

	return map[string]string{
		"{{fullName}}":           valueOrDash(req.Personal.FullName),
		"{{panNumber}}":          valueOrDash(strings.ToUpper(req.Personal.PANNumber)),
		"{{dob}}":                formatPDFDate(req.Personal.DOB),
		"{{phone}}":              valueOrDash(req.Personal.Phone),
		"{{email}}":              valueOrDash(req.Personal.Email),
		"{{fullAddress}}":        valueOrDash(req.Personal.FullAddress),
		"{{startDate}}":          formatTimePtrPDF(req.Agreement.StartDate),
		"{{lendersList}}":        lendersList,
		"{{totalOutstanding}}":   "Rs. " + formatIndianNumber(totalOutstanding),
		"{{duration}}":           formatDurationDisplay(req.Agreement.Duration),
		"{{consultingFees}}":     "Rs. " + formatIndianNumber(req.Agreement.ConsultingFees),
		"{{totalConsultingFee}}": "Rs. " + formatIndianNumber(totalFee),
		"{{monthlyIncome}}":      "Rs. " + formatIndianNumber(req.Agreement.MonthlyIncome),
		"{{executedDate}}":       formatTimePtrPDF(req.Agreement.ExecutedDate),
		"{{preparedBy}}":         valueOrDash(preparedBy),
	}
}

func formatDurationDisplay(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "-"
	}
	if _, err := strconv.Atoi(s); err == nil {
		if s == "1" {
			return "1 Month"
		}
		return s + " Months"
	}
	return s
}

func formatIndianNumber(amount float64) string {
	n := int64(amount + 0.5)
	if n < 0 {
		return "-" + formatIndianNumber(-float64(n))
	}
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	lastThree := s[len(s)-3:]
	rest := s[:len(s)-3]
	var parts []string
	for len(rest) > 2 {
		parts = append([]string{rest[len(rest)-2:]}, parts...)
		rest = rest[:len(rest)-2]
	}
	if rest != "" {
		parts = append([]string{rest}, parts...)
	}
	return strings.Join(parts, ",") + "," + lastThree
}

func valueOrDash(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	return s
}

func formatTimePtrPDF(t *time.Time) string {
	if t == nil || t.IsZero() {
		return "-"
	}
	return t.Local().Format("2006-01-02")
}

func formatPDFDate(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "-"
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.Local().Format("2006-01-02")
	}
	if len(raw) >= 10 {
		return raw[:10]
	}
	return raw
}

func sanitizeForLatin1PDF(s string) string {
	replacer := strings.NewReplacer(
		"₹", "Rs. ",
		"—", "-",
		"–", "-",
		"•", "-",
		"’", "'",
		"‘", "'",
		"“", "\"",
		"”", "\"",
		"…", "...",
	)
	return replacer.Replace(s)
}
