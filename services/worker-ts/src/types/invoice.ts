export interface InvoiceItem {
  id: number;
  productId?: number;
  productName: string;
  quantity: number;
  price: number;
  subtotal: number;
}

export interface InvoicePayload {
  orderId: string | number;
  orderNumber?: string;
  customerName: string;
  customerEmail: string;
  customerPhone?: string;
  shippingAddress: string;
  paymentMethod: string;
  paymentStatus: string;
  createdAt: string;
  items: InvoiceItem[];
  subtotal: number;
  discount: number;
  deliveryFee: number;
  total: number;
}

export interface ReportSummaryPayload {
  title: string;
  generatedBy: string;
  startDate: string;
  endDate: string;
  totalRevenue: number;
  totalOrders: number;
  categoryBreakdown: { category: string; count: number; revenue: number }[];
}
