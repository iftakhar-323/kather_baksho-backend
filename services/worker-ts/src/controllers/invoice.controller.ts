import { Request, Response } from 'express';
import PDFDocument from 'pdfkit';
import { InvoicePayload } from '../types/invoice';

export const generateInvoicePDF = (req: Request, res: Response): void => {
  try {
    const payload: InvoicePayload = req.body;

    if (!payload.orderId || !payload.items || !Array.isArray(payload.items)) {
      res.status(400).json({
        error: 'Invalid invoice payload. orderId and items array are required.'
      });
      return;
    }

    const doc = new PDFDocument({ margin: 50, size: 'A4' });

    // Set response headers for direct inline viewing or download
    res.setHeader('Content-Type', 'application/pdf');
    res.setHeader(
      'Content-Disposition',
      `inline; filename="invoice_order_${payload.orderId}.pdf"`
    );
    res.setHeader('X-Generated-By', 'kather_baksho-worker-ts');

    doc.pipe(res);

    // Color Palette
    const primaryColor = '#1e3d29'; // Botanical Forest Green
    const secondaryColor = '#2d6a4f'; // Leaf Green
    const textColor = '#333333';
    const lightGray = '#f5f7f5';
    const borderColor = '#d8e2dc';

    // Header Background Accent
    doc.rect(0, 0, doc.page.width, 12).fill(primaryColor);

    // Company Header
    doc
      .fillColor(primaryColor)
      .fontSize(22)
      .font('Helvetica-Bold')
      .text('KATHER BAKSHO', 50, 40)
      .fontSize(9)
      .font('Helvetica')
      .fillColor(secondaryColor)
      .text('Botanical Nursery, Terrariums & Premium Wooden Decor', 50, 66)
      .fillColor(textColor)
      .fontSize(8)
      .text('House 42, Road 11, Banani / Mirpur Botanical Hub, Dhaka, Bangladesh', 50, 78)
      .text('Web: http://localhost:8082 | Email: support@kather_baksho.com', 50, 90);

    // Invoice Meta Right-Aligned
    const invNumber = payload.orderNumber || `INV-${String(payload.orderId).padStart(6, '0')}`;
    doc
      .fontSize(18)
      .font('Helvetica-Bold')
      .fillColor(primaryColor)
      .text('TAX INVOICE', 350, 40, { align: 'right' })
      .fontSize(9)
      .font('Helvetica')
      .fillColor(textColor)
      .text(`Invoice No: ${invNumber}`, 350, 64, { align: 'right' })
      .text(`Date: ${new Date(payload.createdAt || Date.now()).toLocaleDateString()}`, 350, 76, { align: 'right' })
      .text(`Payment: ${payload.paymentMethod?.toUpperCase() || 'COD'} (${payload.paymentStatus || 'PAID'})`, 350, 88, { align: 'right' });

    doc.moveTo(50, 110).lineTo(545, 110).strokeColor(borderColor).lineWidth(1).stroke();

    // Bill To & Shipping Information
    doc
      .fontSize(10)
      .font('Helvetica-Bold')
      .fillColor(primaryColor)
      .text('BILLED & DELIVERED TO:', 50, 125);

    doc
      .fontSize(9)
      .font('Helvetica')
      .fillColor(textColor)
      .text(`Customer Name: ${payload.customerName || 'Customer'}`, 50, 140)
      .text(`Email: ${payload.customerEmail || 'N/A'}`, 50, 153)
      .text(`Phone: ${payload.customerPhone || 'N/A'}`, 50, 166)
      .text(`Delivery Address: ${payload.shippingAddress || 'Standard Nursery Delivery'}`, 50, 179, { width: 300 });

    doc.moveTo(50, 210).lineTo(545, 210).strokeColor(borderColor).lineWidth(1).stroke();

    // Table Header
    const tableTop = 225;
    doc.rect(50, tableTop, 495, 20).fill(lightGray);

    doc
      .fillColor(primaryColor)
      .fontSize(9)
      .font('Helvetica-Bold')
      .text('#', 60, tableTop + 5)
      .text('Item Description', 90, tableTop + 5)
      .text('Qty', 330, tableTop + 5, { align: 'center', width: 40 })
      .text('Unit Price', 380, tableTop + 5, { align: 'right', width: 70 })
      .text('Subtotal (BDT)', 460, tableTop + 5, { align: 'right', width: 75 });

    let currentY = tableTop + 25;

    payload.items.forEach((item, index) => {
      const linePrice = item.price || 0;
      const lineSubtotal = item.subtotal || (linePrice * item.quantity);

      doc
        .font('Helvetica')
        .fontSize(8.5)
        .fillColor(textColor)
        .text(String(index + 1), 60, currentY)
        .text(item.productName || `Botanical Item #${item.productId}`, 90, currentY, { width: 230 })
        .text(String(item.quantity), 330, currentY, { align: 'center', width: 40 })
        .text(`BDT ${linePrice.toFixed(2)}`, 380, currentY, { align: 'right', width: 70 })
        .text(`BDT ${lineSubtotal.toFixed(2)}`, 460, currentY, { align: 'right', width: 75 });

      currentY += 20;
      doc.moveTo(50, currentY - 5).lineTo(545, currentY - 5).strokeColor('#f0f0f0').lineWidth(0.5).stroke();
    });

    // Summary Box
    const summaryTop = Math.max(currentY + 15, 450);
    const summaryX = 350;

    doc
      .fontSize(9)
      .font('Helvetica')
      .fillColor(textColor)
      .text('Subtotal:', summaryX, summaryTop)
      .text(`BDT ${(payload.subtotal || 0).toFixed(2)}`, 450, summaryTop, { align: 'right', width: 85 })

      .text('Discount:', summaryX, summaryTop + 16)
      .text(`- BDT ${(payload.discount || 0).toFixed(2)}`, 450, summaryTop + 16, { align: 'right', width: 85 })

      .text('Delivery Fee:', summaryX, summaryTop + 32)
      .text(`BDT ${(payload.deliveryFee || 0).toFixed(2)}`, 450, summaryTop + 32, { align: 'right', width: 85 });

    doc.moveTo(summaryX, summaryTop + 48).lineTo(545, summaryTop + 48).strokeColor(primaryColor).lineWidth(1).stroke();

    doc
      .fontSize(11)
      .font('Helvetica-Bold')
      .fillColor(primaryColor)
      .text('Grand Total:', summaryX, summaryTop + 54)
      .text(`BDT ${(payload.total || 0).toFixed(2)}`, 440, summaryTop + 54, { align: 'right', width: 95 });

    // Official Seal / Authenticity Note
    doc
      .rect(50, summaryTop, 260, 65)
      .strokeColor(borderColor)
      .lineWidth(1)
      .stroke();

    doc
      .fontSize(8)
      .font('Helvetica-Bold')
      .fillColor(secondaryColor)
      .text('DIGITALLY CERTIFIED & VERIFIED', 60, summaryTop + 10)
      .font('Helvetica')
      .fillColor('#666666')
      .text('Generated by Kather Baksho TypeScript Microservice', 60, summaryTop + 24)
      .text(`Order Hash: ${Buffer.from(String(payload.orderId) + (payload.customerEmail || '')).toString('base64').substring(0, 20)}`, 60, summaryTop + 36)
      .text('Thank you for planting greenery and supporting artisanal woodworking!', 60, summaryTop + 48, { width: 240 });

    // Footer
    doc
      .fontSize(7.5)
      .fillColor('#999999')
      .text('Questions? Contact us at support@kather_baksho.com or visit http://localhost:8082', 50, 750, { align: 'center', width: 495 });

    doc.end();
  } catch (err: any) {
    console.error('[Worker-TS] Invoice Generation Error:', err);
    res.status(500).json({ error: 'Failed to generate PDF invoice', details: err.message });
  }
};
