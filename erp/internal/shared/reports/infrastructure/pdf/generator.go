// Package pdf implementa domain.Renderer produciendo PDF desde HTML usando chromedp.
// Requiere Google Chrome o Chromium instalado en el sistema.
// En Linux de producción: apt-get install -y chromium-browser
package pdf

import (
	"context"
	"fmt"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"

	"github.com/diegofxm/erp/internal/shared/reports/domain"
	"github.com/diegofxm/erp/internal/shared/reports/infrastructure/html"
)

// Generator implementa domain.Renderer produciendo PDF.
type Generator struct {
	htmlRenderer *html.Renderer
}

func NewGenerator(r *html.Renderer) *Generator {
	return &Generator{htmlRenderer: r}
}

var _ domain.Renderer = (*Generator)(nil)

func (g *Generator) Render(ctx context.Context, req domain.RenderRequest) ([]byte, error) {
	if req.Format != domain.FormatPDF {
		return nil, fmt.Errorf("reports/pdf: formato no soportado: %s", req.Format)
	}

	htmlContent, err := g.htmlRenderer.RenderToString(req.TemplateID, req.Data)
	if err != nil {
		return nil, err
	}

	return htmlToPDF(ctx, htmlContent)
}

func htmlToPDF(ctx context.Context, htmlContent string) ([]byte, error) {
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx,
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-dev-shm-usage", true),
		)...,
	)
	defer cancelAlloc()

	chromeCtx, cancelChrome := chromedp.NewContext(allocCtx)
	defer cancelChrome()

	var pdfBuf []byte
	if err := chromedp.Run(chromeCtx,
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			frameTree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(frameTree.Frame.ID, htmlContent).Do(ctx)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).  // A4 ancho en pulgadas
				WithPaperHeight(11.69). // A4 alto
				WithMarginTop(0.4).
				WithMarginBottom(0.4).
				WithMarginLeft(0.4).
				WithMarginRight(0.4).
				Do(ctx)
			if err != nil {
				return err
			}
			pdfBuf = buf
			return nil
		}),
	); err != nil {
		return nil, fmt.Errorf("reports/pdf: chromedp: %w", err)
	}
	return pdfBuf, nil
}
