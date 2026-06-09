// api-endpoints.ts — Tập trung tất cả API endpoint bên ngoài dùng trong mail-provider stock check
// Khi provider thay đổi domain, chỉ sửa ở đây — không phải trong component.

export const MAIL_PROVIDER_ENDPOINTS = {
  /** ZeusX instock: GET, trả về { Code: 0, Data: ZeusXItem[] } */
  zeusX: 'https://api.zeus-x.ru/instock',

  /** Mail30s products: GET ?api_key=... — trả về { success: bool, data: Mail30sProduct[] } */
  mail30sProducts: (apiKey: string) =>
    `https://api.mailotp.com/api/automation/products?api_key=${encodeURIComponent(apiKey)}&min_stock=0`,

  /** Store1s products: GET ?api_key=... — trả về { status: 'success', categories: [...] } */
  store1sProducts: (apiKey: string) =>
    `https://store1s.com/api/products.php?api_key=${encodeURIComponent(apiKey)}`,

  /** DongVanFB account types: GET ?apikey=... — trả về { status: bool, data: DvfbItem[] } */
  dongvanfbAccountTypes: (apiKey: string) =>
    `https://api.dongvanfb.net/user/account_type?apikey=${encodeURIComponent(apiKey)}`,
} as const
