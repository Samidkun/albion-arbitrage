# AlbionFlow — Real-Time Albion Online Market & Refining Arbitrage Engine
## Design Specification

> **Status:** Draft / Ready for Review · **Tanggal:** 2026-09-14 · **Pemilik:** Samid (Dimas Sadewa)  
> **Klasifikasi:** Architectural (Project Baru di `/mnt/data/01_Projects/Porto/albion-arbitrage`)  
> **Tujuan:** High-precision data pipeline & decision engine untuk maksimalkan Silver/Focus dan profit arbitrase lintas kota di Albion Online tanpa risiko data usang.

---

## 1. Latar Belakang & Masalah Fatal

Di Albion Online, ekonomi digerakkan 100% oleh pemain (*player-driven economy*). Margin keuntungan dari **refining** (mengolah bahan mentah jadi balok/batangan/kain/kulit) dan **transport arbitrage** (beli murah di Kota A, jual mahal di Kota B/Caerleon) bernilai jutaan Silver per hari.

Namun, mengolah data pasar memiliki **risiko fatal**:
1. **Stale Data Trap:** Jika harga diambil dari snapshot yang sudah berumur 6–12 jam lalu, pemain bisa membawa modal puluhan juta Silver melintasi red/yellow zone dan mendapati harga sudah anjlok di kota tujuan.
2. **Hierarchical Crafting Mistake:** Mengolah bahan T5–T8 tidak hanya butuh bahan mentah setara, tapi juga **1 unit refined material dari tier sebelumnya** (contoh: 1 unit T6 Bar butuh 3 T6 Ore + 1 T5 Bar). Jika kalkulator mengabaikan harga atau return rate pada tier sebelumnya, perhitungan profit 100% salah.
3. **Station Fee & Nutrition Variance:** Pemilik toko pengrajin di kota menetapkan pajak nutrisi yang fluktuatif (misal 500 s/d 1800 silver per 100 nutrisi).
4. **Market Tax & Order Fees:** Pemain non-premium terkena pajak 8% sedangkan premium 4%, ditambah biaya pasang order 2.5%. Salah menghitung pajak membuat transaksi yang tampak untung menjadi buntung.

**Objektif:** Membangun engine berbasis web (*Astro + Go + SQLite*) dengan filter ketat terhadap kesegaran data (*data age guard*), audit trail perhitungan matematis transparan (*show the math*), dan presisi formula 100% cocok dengan mekanika game.

---

## 2. Pilihan Arsitektur & Komponen

```
┌────────────────────────────────────────────────────────┐
│ The Albion Online Data Project (AODP) Public REST API  │
│ Regions: Asia (Singapore), Americas (US), Europe (EU)  │
└───────────────────────────┬────────────────────────────┘
                            │ (HTTP GET with rate-limiter & cached ETags)
                            ▼
┌────────────────────────────────────────────────────────┐
│ Go Backend Engine (localhost:8081)                     │
│ ├── Ingestion Worker (Polls 5 commodities T4-T8)       │
│ ├── Data Freshness Guard (Rejects/flags stale data)    │
│ ├── SQLite Store (data/market.db with WAL mode)        │
│ ├── Refining Engine (Hierarchy, RRR, Nutrition, Focus) │
│ ├── Arbitrage Scanner (6 Royal Cities + Black Market)  │
│ └── HTTP JSON API (Chi router)                         │
└───────────────────────────┬────────────────────────────┘
                            │ (JSON REST / SSE Stream)
                            ▼
┌────────────────────────────────────────────────────────┐
│ Astro Web Dashboard (localhost:4321)                   │
│ ├── Region Switcher (Asia / Americas / Europe)         │
│ ├── Freshness Indicator (<15m Green, >60m Stale Red)   │
│ ├── Refining Leaderboard (Sort by Silver per Focus)    │
│ ├── Hauling Arbitrage Matrix (Route & Net Margin)      │
│ └── "Show the Math" Audit Modal (Step-by-step breakdown)│
└────────────────────────────────────────────────────────┘
```

---

## 3. Aturan Presisi Formula & Anti-Silver-Loss

### 3.1 Hierarki Bahan Refining (Resep Resmi Game)
Setiap 1 unit output refined material membutuhkan:
- **Tier 4 (T4):** 2x T4 Raw + 1x T3 Refined
- **Tier 5 (T5):** 3x T5 Raw + 1x T4 Refined (Catatan: enchant level .1/.2/.3/.4 membutuhkan T4 refined ber-enchant setara!)
- **Tier 6 (T6):** 4x T6 Raw + 1x T5 Refined
- **Tier 7 (T7):** 5x T7 Raw + 1x T6 Refined
- **Tier 8 (T8):** 5x T8 Raw + 1x T7 Refined

### 3.2 Resource Return Rate (RRR)
Di Albion, RRR mengembalikan sebagian bahan mentah dan bahan refined sebelumnya ke kantong pemain:
$$\text{Effective Quantity Used} = \text{Required Quantity} \times (1 - \text{RRR})$$

| Lokasi & Kondisi | RRR Standar | Efektif Penghematan Bahan |
|---|---|---|
| **Kota Biasa (No Bonus)** | 15.2% | -15.2% |
| **Kota dengan Spesialisasi Biome** | 36.7% | -36.7% (setara ~57.9% bonus output) |
| **Kota Spesialisasi + Crafting Focus** | 53.9% | -53.9% (setara ~116.9% bonus output) |

**Bonus Statis Kota Resmi:**
- **Fort Sterling:** Wood (Kayu)
- **Thetford:** Ore (Biji Besi)
- **Lymhurst:** Fiber (Serat)
- **Bridgewatch:** Stone (Batu)
- **Martlock:** Hide (Kulit)

### 3.3 Biaya Nutrisi Toko Tempa (Station Nutrition Fee)
Setiap crafting mengonsumsi nutrisi berdasarkan `Item Value` dasar produk:
$$\text{Nutrition Consumed} = \text{Item Value} \times 0.1125$$
$$\text{Crafting Fee (Silver)} = \frac{\text{Nutrition Consumed} \times \text{User Set Station Fee}}{100}$$
*Pengguna dapat mengubah slider Station Fee di UI (default: 600 Silver per 100 Nutrition).*

### 3.4 Silver Per Focus (SPF)
Indikator utama pemain apakah layak membakar 10.000 focus harian:
$$\text{SPF} = \frac{\text{Net Profit (dengan Focus)} - \text{Net Profit (tanpa Focus)}}{\text{Focus Points Digunakan}}$$

### 3.5 Arbitrase Antar-Kota (Hauling) & Pemotongan Pajak
$$\text{Net Revenue} = \text{Sell Price} \times (1 - \text{Market Tax}) - (\text{Sell Price} \times \text{Setup Fee})$$
- `Market Tax`: 4% jika Premium, 8% jika Non-Premium.
- `Setup Fee`: 2.5% jika jual via Sell Order (0% jika langsung jual ke Buy Order instan).
- `Net Margin`:
$$\text{Net Profit} = \text{Net Revenue} - \text{Buy Price}$$

---

## 4. Perlindungan Validitas Data (Data Guardrails)

Untuk mencegah keputusan fatal akibat data busuk:
1. **Age Threshold Guard:**
   - **Fresh (< 30 menit):** Hijau. Rekomendasi aktif.
   - **Aging (30 – 60 menit):** Kuning/Amber. Peringatan: *"Cek ulang harga di pasar game sebelum borong banyak"*.
   - **Stale (> 60 menit):** Merah. Dikeluarkan dari tabel rekomendasi utama (diberi flag `STALE_DATA`).
2. **Zero Price Filtering:**
   Jika `sell_price_min == 0` atau tidak ada order aktif, item langsung di-skip dan tidak dianggap seharga 0 (menghindari false-positive profit tak hingga).
3. **"Show the Math" Drawer:**
   Setiap baris tabel memiliki tombol `[Rincian Kalkulasi]` yang menampilkan nota hitungan per unit:
   - Harga beli Raw Material (Nama kota & timestamp harga)
   - Harga beli Previous Refined Material (Nama kota & timestamp harga)
   - Diskon RRR yang didapat
   - Ongkos sewa tukang tempa
   - Pajak pasar yang akan dipotong
   - Estimasi profit bersih per unit & total per batch.

---

## 5. Rencana Modul Teknis

### 5.1 Backend Go (`backend/`)
- `pkg/aodp`: Client HTTP untuk endpoint The Albion Online Data Project:
  `https://west.albion-online-data.com/api/v2/stats/prices/...` (Americas)  
  `https://east.albion-online-data.com/api/v2/stats/prices/...` (Asia)  
  `https://europe.albion-online-data.com/api/v2/stats/prices/...` (Europe)
- `pkg/engine`: Logika bisnis kalkulasi RRR, SPF, dan Arbitrage Route Finder.
- `pkg/store`: SQLite embedded engine (`price_snapshots`, `item_metadata`).
- `cmd/server`: HTTP API Chi router menyajikan endpoint:
  - `GET /api/v1/refining/matrix?region=asia&tax=600&premium=true`
  - `GET /api/v1/arbitrage/routes?region=asia&min_profit=1000`
  - `POST /api/v1/prices/refresh?region=asia`

### 5.2 Frontend Astro (`frontend/`)
- Astro 5 SSR / Static dengan endpoint fetch lokal.
- Design tokens Anti-UI-Slop: Clean dark `#16181C`, hairline borders `#2A2E35`, electric lime `#C7F53B` untuk profit terbaik.
- Interaktivitas:
  - Tab Region: `Asia`, `Europe`, `Americas`.
  - Tab Komoditas: `Wood`, `Ore`, `Hide`, `Fiber`, `Stone`.
  - Filter: Tier (`T4` - `T8`), Enchant (`.0` - `.4`), Status Premium (On/Off).
  - Slider Station Tax (Silver / 100 nutrition).

---

## 6. Verifikasi & Pengujian

1. **Unit Tests Formula Matematika:**
   - Memastikan pengembalian bahan RRR 36.7% dan 53.9% menghasilkan nominal desimal akurat.
   - Memastikan perhitungan T5-T8 mengikutsertakan harga tier sebelumnya.
   - Memastikan perhitungan SPF akurat.
2. **Ingest Guard Tests:**
   - Memastikan data berumur > 60 menit ditandai `stale`.
   - Memastikan item dengan harga 0 tidak menghasilkan anomali profit.
3. **E2E Integration Test:**
   - Simulasi snapshot pasar dari 6 kota dengan data riil untuk memverifikasi keluaran rute arbitrase yang logis.
