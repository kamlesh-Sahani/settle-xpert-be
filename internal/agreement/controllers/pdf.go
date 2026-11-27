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
	"strconv"
	"strings"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/agreement/models"
	leadmodels "lead-crm/backend/internal/lead/models"
	"lead-crm/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-pdf/fpdf"
	"github.com/kamlesh-Sahani/go-commons/response"
	commonsutils "github.com/kamlesh-Sahani/go-commons/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	xhtml "golang.org/x/net/html"
)

const (
	pageMarginL  = 18.0
	pageMarginR  = 18.0
	pageContentW = 210.0 - pageMarginL - pageMarginR // 174mm
	pageTopY     = 34.0
)

var (
	htmlTagRe = regexp.MustCompile(`<[^>]*>`)
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
	text      string
	bold      bool
	italic    bool
	underline bool
	color     *rgbColor
}

type pdfFontConfig struct {
	family  string
	unicode bool
}

type GenerateAgreementPDFRequest struct {
	LeadID string `json:"leadId" binding:"required,max=24"`
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

func cleanText(cfg pdfFontConfig, s string) string {
	if cfg.unicode {
		return s
	}
	return sanitizeForLatin1PDF(s)
}

func stripHTMLTags(s string) string {
	return htmlTagRe.ReplaceAllString(s, " ")
}

// GeneratePDF renders the agreement PDF for a lead using saved lead data and DB agreement templates.
func GeneratePDF(c *gin.Context) {
	var rawReq GenerateAgreementPDFRequest
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		response.BadRequest(c, commonsutils.FormatValidationError(err))
		return
	}

	leadObjID, err := primitive.ObjectIDFromHex(strings.TrimSpace(rawReq.LeadID))
	if err != nil || leadObjID.IsZero() {
		response.BadRequest(c, "Invalid lead ID format")
		return
	}

	leadCol := database.GetCollection("leads")
	tplCol := database.GetCollection(models.AgreementTemplateCollection)
	if leadCol == nil || tplCol == nil {
		response.InternalServerError(c, "Database service unavailable")
		return
	}

	ctx := c.Request.Context()
	var existing leadmodels.Lead
	if err := leadCol.FindOne(
		ctx,
		bson.M{"_id": leadObjID, "deleted_at": bson.M{"$exists": false}},
	).Decode(&existing); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			response.NotFound(c, "Lead not found")
			return
		}
		log.Printf("[ERROR] [AgreementPDF] FindOne lead failed: %v", err)
		response.InternalServerError(c, "Failed to retrieve lead")
		return
	}

	var tpl models.AgreementTemplate
	if err := tplCol.FindOne(
		ctx,
		bson.M{"key": models.DefaultAgreementTemplateKey},
	).Decode(&tpl); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			response.BadRequest(c, "Agreement template not found. Please save a template first.")
			return
		}
		log.Printf("[ERROR] [AgreementPDF] FindOne template failed: %v", err)
		response.InternalServerError(c, "Failed to retrieve agreement template")
		return
	}
	if strings.TrimSpace(stripHTMLTags(tpl.Body)) == "" {
		response.BadRequest(c, "Agreement body template cannot be empty")
		return
	}

	var lenders []leadmodels.Lender
	var monthlyIncome float64
	if existing.Financial != nil {
		lenders = existing.Financial.Lenders
		monthlyIncome = existing.Financial.MonthlyIncome
	}

	var agreementDetails AgreementDetailsInput
	agreementDetails.MonthlyIncome = monthlyIncome
	preparedBy := ""
	if existing.Agreement != nil {
		agreementDetails.Duration = existing.Agreement.Duration
		agreementDetails.ResolutionDuration = existing.Agreement.ResolutionDuration
		agreementDetails.ConsultingFees = existing.Agreement.ConsultingFees
		agreementDetails.StartDate = existing.Agreement.StartDate
		agreementDetails.ExecutedDate = existing.Agreement.ExecutedDate
		preparedBy = strings.TrimSpace(existing.Agreement.PreparedByName)
	}
	if preparedBy == "" {
		if emp, ok := utils.GetAuthEmployee(c); ok && emp != nil {
			preparedBy = emp.Name
		}
	}

	now := time.Now().Local()
	if agreementDetails.ExecutedDate == nil || agreementDetails.ExecutedDate.IsZero() {
		agreementDetails.ExecutedDate = &now
	}
	if agreementDetails.StartDate == nil || agreementDetails.StartDate.IsZero() {
		agreementDetails.StartDate = agreementDetails.ExecutedDate
	}

	req := CreateAgreementRequest{
		LeadID:  rawReq.LeadID,
		Lenders: lenders,
		Personal: AgreementPersonalInput{
			FullName:    existing.Personal.FullName,
			PANNumber:   existing.Personal.PANNumber,
			DOB:         existing.Personal.DOB,
			Phone:       existing.Personal.Phone,
			Email:       existing.Personal.Email,
			FullAddress: existing.Personal.FullAddress,
		},
		Agreement: agreementDetails,
	}

	buf, err := buildAgreementPDFBytes(tpl.Body, tpl.Footer, req, preparedBy)
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
	c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/pdf", buf)
}

func buildAgreementPDFBytes(templateBody, templateFooter string, req CreateAgreementRequest, preparedBy string) ([]byte, error) {
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
		renderPDFFooter(pdf, fontCfg, templateFooter, replacements)
	})

	pdf.AddPage()
	renderHTMLAgreementBody(pdf, fontCfg, templateBody, replacements, req)

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
	pdf.SetXY(rightX-totalW, y)
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

func parseWrappedHTML(raw string) *xhtml.Node {
	doc, err := xhtml.Parse(strings.NewReader("<div>" + raw + "</div>"))
	if err != nil {
		return nil
	}
	var root *xhtml.Node
	var findRoot func(*xhtml.Node)
	findRoot = func(n *xhtml.Node) {
		if n.Type == xhtml.ElementNode && n.Data == "div" {
			root = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findRoot(c)
			if root != nil {
				return
			}
		}
	}
	findRoot(doc)
	return root
}

func extractNodeAlign(n *xhtml.Node, fallback string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, "style") {
			low := strings.ToLower(a.Val)
			if strings.Contains(low, "text-align") {
				if strings.Contains(low, "center") {
					return "C"
				}
				if strings.Contains(low, "right") {
					return "R"
				}
				if strings.Contains(low, "justify") {
					return "J"
				}
				if strings.Contains(low, "left") {
					return "L"
				}
			}
		}
	}
	return fallback
}

func replacePlaceholders(cfg pdfFontConfig, text string, replacements map[string]string, pageNo int) string {
	out := text
	if pageNo > 0 {
		out = strings.ReplaceAll(out, "{{page}}", strconv.Itoa(pageNo))
		out = strings.ReplaceAll(out, "{{totalPages}}", "{nb}")
	}
	for token, val := range replacements {
		if strings.Contains(out, token) {
			out = strings.ReplaceAll(out, token, val)
		}
	}
	return cleanText(cfg, out)
}

func collectInlineHTML(cfg pdfFontConfig, n *xhtml.Node, b, it, u bool, defaultColor *rgbColor, replacements map[string]string, pageNo int, out *[]textSegment) {
	switch n.Type {
	case xhtml.TextNode:
		txt := strings.ReplaceAll(n.Data, "\u00a0", " ")
		if txt == "" {
			return
		}
		*out = append(*out, textSegment{
			text:      replacePlaceholders(cfg, txt, replacements, pageNo),
			bold:      b,
			italic:    it,
			underline: u,
			color:     defaultColor,
		})
	case xhtml.ElementNode:
		nb, nit, nu := b, it, u
		switch strings.ToLower(n.Data) {
		case "strong", "b":
			nb = true
		case "em", "i":
			nit = true
		case "u":
			nu = true
		case "br":
			*out = append(*out, textSegment{text: "\n"})
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			collectInlineHTML(cfg, c, nb, nit, nu, defaultColor, replacements, pageNo, out)
		}
	}
}

func renderPDFFooter(pdf *fpdf.Fpdf, fontCfg pdfFontConfig, templateFooter string, replacements map[string]string) {
	footerPlain := strings.TrimSpace(stripHTMLTags(templateFooter))
	if footerPlain == "" {
		return
	}

	pdf.SetY(-15)
	pdf.SetDrawColor(colorBorder.r, colorBorder.g, colorBorder.b)
	pdf.SetLineWidth(0.25)
	pdf.Line(pageMarginL, 281, 210.0-pageMarginR, 281)
	pdf.SetY(282.5)

	root := parseWrappedHTML(templateFooter)
	if root == nil {
		pdf.SetFont(fontCfg.family, "", 7.8)
		pdf.SetTextColor(colorSlate500.r, colorSlate500.g, colorSlate500.b)
		pdf.CellFormat(pageContentW, 4.4, replacePlaceholders(fontCfg, footerPlain, replacements, pdf.PageNo()), "", 0, "C", false, 0, "")
		return
	}

	drawFooterLine := func(segs []textSegment, align string) {
		if len(segs) == 0 || pdf.GetY() > 292 {
			return
		}
		prevMargin := pdf.GetCellMargin()
		pdf.SetCellMargin(0)
		defer pdf.SetCellMargin(prevMargin)

		type span struct {
			text  string
			style string
			width float64
		}
		var spans []span
		totalW := 0.0
		for _, seg := range segs {
			style := ""
			if seg.bold {
				style += "B"
			}
			if seg.italic {
				style += "I"
			}
			if seg.underline {
				style += "U"
			}
			pdf.SetFont(fontCfg.family, style, 7.8)
			w := pdf.GetStringWidth(seg.text)
			spans = append(spans, span{text: seg.text, style: style, width: w})
			totalW += w
		}

		if len(spans) == 1 && (align == "C" || align == "R" || align == "L") {
			pdf.SetX(pageMarginL)
			pdf.SetFont(fontCfg.family, spans[0].style, 7.8)
			pdf.SetTextColor(colorSlate500.r, colorSlate500.g, colorSlate500.b)
			pdf.CellFormat(pageContentW, 4.2, spans[0].text, "", 1, align, false, 0, "")
			return
		}

		startX := pageMarginL
		if align == "C" && pageContentW > totalW {
			startX = pageMarginL + (pageContentW-totalW)/2
		} else if align == "R" && pageContentW > totalW {
			startX = pageMarginL + (pageContentW - totalW)
		}

		pdf.SetX(startX)
		pdf.SetTextColor(colorSlate500.r, colorSlate500.g, colorSlate500.b)
		for _, sp := range spans {
			pdf.SetFont(fontCfg.family, sp.style, 7.8)
			pdf.CellFormat(sp.width, 4.2, sp.text, "", 0, "L", false, 0, "")
		}
		pdf.Ln(4.2)
	}

	var walkFooter func(*xhtml.Node)
	walkFooter = func(n *xhtml.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != xhtml.ElementNode {
				continue
			}
			switch strings.ToLower(c.Data) {
			case "p", "h1", "h2", "h3", "blockquote", "div":
				align := extractNodeAlign(c, "C")
				var segs []textSegment
				collectInlineHTML(fontCfg, c, false, false, false, &colorSlate500, replacements, pdf.PageNo(), &segs)
				var cur []textSegment
				for _, s := range segs {
					if s.text == "\n" {
						drawFooterLine(cur, align)
						cur = nil
						continue
					}
					cur = append(cur, s)
				}
				if len(cur) > 0 {
					drawFooterLine(cur, align)
				}
			default:
				walkFooter(c)
			}
		}
	}
	walkFooter(root)
}

func renderHTMLAgreementBody(pdf *fpdf.Fpdf, fontCfg pdfFontConfig, htmlBody string, replacements map[string]string, req CreateAgreementRequest) {
	root := parseWrappedHTML(htmlBody)
	if root == nil {
		txt := replacePlaceholders(fontCfg, strings.TrimSpace(stripHTMLTags(htmlBody)), replacements, 0)
		renderRichWrappedTextAligned(pdf, fontCfg, pageMarginL, pageContentW, 5.4, 10, "L", []textSegment{
			{text: txt, color: &colorSlate800},
		})
		return
	}

	plainFromSegs := func(segs []textSegment) string {
		var sb strings.Builder
		for _, s := range segs {
			sb.WriteString(s.text)
		}
		return strings.TrimSpace(sb.String())
	}

	renderBlockSegs := func(x, w, lineH, size float64, align string, segs []textSegment) {
		var current []textSegment
		for _, seg := range segs {
			if seg.text == "\n" {
				if len(current) > 0 {
					renderRichWrappedTextAligned(pdf, fontCfg, x, w, lineH, size, align, current)
					current = nil
				} else {
					pdf.Ln(lineH * 0.6)
				}
				continue
			}
			current = append(current, seg)
		}
		if len(current) > 0 {
			renderRichWrappedTextAligned(pdf, fontCfg, x, w, lineH, size, align, current)
		}
	}

	var walkBlocks func(*xhtml.Node)
	walkBlocks = func(n *xhtml.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != xhtml.ElementNode {
				continue
			}
			tag := strings.ToLower(c.Data)
			align := extractNodeAlign(c, "L")
			switch tag {
			case "h1", "h2", "h3":
				var segs []textSegment
				collectInlineHTML(fontCfg, c, true, false, false, &colorSlate900, replacements, 0, &segs)
				if plainFromSegs(segs) == "" {
					continue
				}
				if pdf.GetY() > 252 {
					pdf.AddPage()
				}
				pdf.Ln(2.5)
				size := 10.8
				if tag == "h1" {
					size = 12.8
				} else if tag == "h2" {
					size = 11.8
				}
				renderBlockSegs(pageMarginL, pageContentW, 5.8, size, align, segs)
				pdf.Ln(1.5)

			case "hr":
				if pdf.GetY() > 268 {
					pdf.AddPage()
				}
				pdf.Ln(2)
				y := pdf.GetY()
				pdf.SetDrawColor(colorBorder.r, colorBorder.g, colorBorder.b)
				pdf.SetLineWidth(0.35)
				pdf.Line(pageMarginL, y, 210.0-pageMarginR, y)
				pdf.Ln(3)

			case "ul", "ol":
				idx := 0
				for li := c.FirstChild; li != nil; li = li.NextSibling {
					if li.Type != xhtml.ElementNode || strings.ToLower(li.Data) != "li" {
						continue
					}
					idx++
					var segs []textSegment
					collectInlineHTML(fontCfg, li, false, false, false, &colorSlate800, replacements, 0, &segs)
					if plainFromSegs(segs) == "" {
						continue
					}
					if pdf.GetY() > 270 {
						pdf.AddPage()
					}
					prefix := cleanText(fontCfg, "•")
					if tag == "ol" {
						prefix = fmt.Sprintf("%d.", idx)
					}
					y := pdf.GetY()
					pdf.SetFont(fontCfg.family, "B", 10)
					pdf.SetTextColor(colorSlate900.r, colorSlate900.g, colorSlate900.b)
					pdf.SetXY(pageMarginL+5, y)
					pdf.CellFormat(5.5, 5.4, prefix, "", 0, "L", false, 0, "")
					renderBlockSegs(pageMarginL+10.5, pageContentW-10.5, 5.4, 10, "L", segs)
					pdf.Ln(1.2)
				}
				pdf.Ln(1)

			case "blockquote":
				var segs []textSegment
				collectInlineHTML(fontCfg, c, false, true, false, &colorSlate600, replacements, 0, &segs)
				if plainFromSegs(segs) == "" {
					continue
				}
				startY := pdf.GetY()
				renderBlockSegs(pageMarginL+4, pageContentW-4, 5.4, 10, align, segs)
				endY := pdf.GetY()
				if endY > startY {
					pdf.SetDrawColor(colorBorder.r, colorBorder.g, colorBorder.b)
					pdf.SetLineWidth(0.8)
					pdf.Line(pageMarginL+1, startY, pageMarginL+1, endY-1)
				}
				pdf.Ln(1.8)

			case "p":
				var rawText strings.Builder
				var extractRaw func(*xhtml.Node)
				extractRaw = func(node *xhtml.Node) {
					if node.Type == xhtml.TextNode {
						rawText.WriteString(node.Data)
					}
					for ch := node.FirstChild; ch != nil; ch = ch.NextSibling {
						extractRaw(ch)
					}
				}
				extractRaw(c)
				switch strings.TrimSpace(rawText.String()) {
				case "{{pageBreak}}":
					pdf.AddPage()
					continue
				case "{{signatureBlock}}":
					renderSignatureBlock(pdf, fontCfg, req.Personal.FullName)
					continue
				case "{{clientInfoTable}}":
					renderClientInfoTable(pdf, fontCfg, req)
					continue
				case "{{lendersTable}}":
					renderLendersTable(pdf, fontCfg, req)
					continue
				case "{{feeScheduleTable}}":
					renderFeeScheduleTable(pdf, fontCfg, req)
					continue
				}

				var segs []textSegment
				collectInlineHTML(fontCfg, c, false, false, false, &colorSlate800, replacements, 0, &segs)
				if plainFromSegs(segs) == "" {
					pdf.Ln(2.5)
					continue
				}
				renderBlockSegs(pageMarginL, pageContentW, 5.4, 10, align, segs)
				pdf.Ln(1.8)

			default:
				walkBlocks(c)
			}
		}
	}

	walkBlocks(root)
}

func renderSignatureBlock(pdf *fpdf.Fpdf, fontCfg pdfFontConfig, clientName string) {
	if pdf.GetY() > 240 {
		pdf.AddPage()
	}
	pdf.Ln(6)
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
	pdf.Ln(3)
}

func renderClientInfoTable(pdf *fpdf.Fpdf, fontCfg pdfFontConfig, req CreateAgreementRequest) {
	if pdf.GetY() > 215 {
		pdf.AddPage()
	}
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
	pdf.Ln(4)
}

func renderLendersTable(pdf *fpdf.Fpdf, fontCfg pdfFontConfig, req CreateAgreementRequest) {
	if pdf.GetY() > 235 {
		pdf.AddPage()
	}
	lCol1 := 64.0
	lCol2 := 60.0
	lCol3 := pageContentW - lCol1 - lCol2

	drawHeader := func() {
		pdf.SetFillColor(colorGreenHdr.r, colorGreenHdr.g, colorGreenHdr.b)
		pdf.SetTextColor(255, 255, 255)
		pdf.SetDrawColor(colorBorder.r, colorBorder.g, colorBorder.b)
		pdf.SetLineWidth(0.25)
		pdf.SetFont(fontCfg.family, "B", 9.5)
		pdf.SetX(pageMarginL)
		pdf.CellFormat(lCol1, 8.6, "   Lender Name*", "1", 0, "L", true, 0, "")
		pdf.CellFormat(lCol2, 8.6, "   Loan Type (PL/CC)", "1", 0, "L", true, 0, "")
		pdf.CellFormat(lCol3, 8.6, "Loan Amt (Balance)   ", "1", 1, "R", true, 0, "")
	}
	drawHeader()

	var totalOutstanding float64
	if len(req.Lenders) == 0 {
		pdf.SetX(pageMarginL)
		pdf.SetTextColor(colorSlate500.r, colorSlate500.g, colorSlate500.b)
		pdf.SetFont(fontCfg.family, "I", 9.5)
		pdf.CellFormat(pageContentW, 8.6, "No lenders selected", "1", 1, "C", false, 0, "")
	} else {
		for _, l := range req.Lenders {
			if pdf.GetY()+8.4 > 276 {
				pdf.AddPage()
				drawHeader()
			}
			pdf.SetX(pageMarginL)
			pdf.SetDrawColor(colorBorder.r, colorBorder.g, colorBorder.b)
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
}

func renderFeeScheduleTable(pdf *fpdf.Fpdf, fontCfg pdfFontConfig, req CreateAgreementRequest) {
	months := 1
	if req.Agreement.Duration > 0 && req.Agreement.Duration <= 60 {
		months = req.Agreement.Duration
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

	if pdf.GetY()+18 > 276 {
		pdf.AddPage()
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
	pdf.Ln(3)
}

func renderRichWrappedTextAligned(pdf *fpdf.Fpdf, fontCfg pdfFontConfig, x, maxW, lineH, fontSize float64, align string, segs []textSegment) {
	type wordToken struct {
		text      string
		bold      bool
		italic    bool
		underline bool
		color     rgbColor
		space     bool
	}

	var tokens []wordToken
	for _, seg := range segs {
		c := colorSlate800
		if seg.color != nil {
			c = *seg.color
		}
		i := 0
		for i < len(seg.text) {
			if seg.text[i] == ' ' || seg.text[i] == '\t' {
				j := i
				for j < len(seg.text) && (seg.text[j] == ' ' || seg.text[j] == '\t') {
					j++
				}
				tokens = append(tokens, wordToken{
					text:      " ",
					bold:      seg.bold,
					italic:    seg.italic,
					underline: seg.underline,
					color:     c,
					space:     true,
				})
				i = j
				continue
			}
			j := i
			for j < len(seg.text) && seg.text[j] != ' ' && seg.text[j] != '\t' {
				j++
			}
			tokens = append(tokens, wordToken{
				text:      seg.text[i:j],
				bold:      seg.bold,
				italic:    seg.italic,
				underline: seg.underline,
				color:     c,
			})
			i = j
		}
	}

	fontStyle := func(t wordToken) string {
		style := ""
		if t.bold {
			style += "B"
		}
		if t.italic {
			style += "I"
		}
		if t.underline {
			style += "U"
		}
		return style
	}

	measureToken := func(t wordToken) float64 {
		pdf.SetFont(fontCfg.family, fontStyle(t), fontSize)
		return pdf.GetStringWidth(t.text)
	}

	drawLine := func(line []wordToken, isLast bool) {
		for len(line) > 0 && line[len(line)-1].space {
			line = line[:len(line)-1]
		}
		if pdf.GetY()+lineH > 276 {
			pdf.AddPage()
		}
		prevMargin := pdf.GetCellMargin()
		pdf.SetCellMargin(0)

		totalW := 0.0
		spaceCount := 0
		widths := make([]float64, len(line))
		for idx, t := range line {
			w := measureToken(t)
			widths[idx] = w
			totalW += w
			if t.space {
				spaceCount++
			}
		}

		startX := x
		extraPerSpace := 0.0
		switch align {
		case "C":
			if maxW > totalW {
				startX = x + (maxW-totalW)/2
			}
		case "R":
			if maxW > totalW {
				startX = x + (maxW - totalW)
			}
		case "J":
			if !isLast && spaceCount > 0 && maxW > totalW {
				extraPerSpace = (maxW - totalW) / float64(spaceCount)
			}
		}

		pdf.SetX(startX)
		for idx, t := range line {
			pdf.SetFont(fontCfg.family, fontStyle(t), fontSize)
			pdf.SetTextColor(t.color.r, t.color.g, t.color.b)
			w := widths[idx]
			if t.space && extraPerSpace > 0 {
				w += extraPerSpace
			}
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
			drawLine(curLine, false)
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
		drawLine(curLine, true)
	}
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
	if req.Agreement.Duration > 0 {
		months = req.Agreement.Duration
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
		"{{resolutionDuration}}": formatDurationDisplay(req.Agreement.ResolutionDuration),
		"{{consultingFees}}":     "Rs. " + formatIndianNumber(req.Agreement.ConsultingFees),
		"{{totalConsultingFee}}": "Rs. " + formatIndianNumber(totalFee),
		"{{monthlyIncome}}":      "Rs. " + formatIndianNumber(req.Agreement.MonthlyIncome),
		"{{executedDate}}":       formatTimePtrPDF(req.Agreement.ExecutedDate),
		"{{preparedBy}}":         valueOrDash(preparedBy),
	}
}

func formatDurationDisplay(months int) string {
	if months <= 0 {
		return "-"
	}
	if months == 1 {
		return "1 Month"
	}
	return fmt.Sprintf("%d Months", months)
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
