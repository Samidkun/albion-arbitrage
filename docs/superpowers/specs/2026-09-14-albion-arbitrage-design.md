# AlbionFlow — Real-Time Albion Online Market & Refining Arbitrage Engine
## Design Specification

> **Status:** Draft / Ready for Review · **Tanggal:** 2026-09-14 · **Pemilik:** Samid (Dimas Sadewa)  
> **Klasifikasi:** Architectural (Project Baru di `/mnt/data/01_Projects/Porto/albion-arbitrage`)  
> **Tujuan:** High-precision data pipeline & decision engine untuk maksimalkan Silver/Focus dan profit arbitrase lintas kota di Albion Online tanpa risiko data usang.

---

## 1. Latar Belakang & Riset Masalah Pemain (Empirical Evidence)

Berdasarkan riset empiris dari keluhan pemain di komunitas (*Reddit r/albiononline*, forum resmi, dan bug reports open-source tools seperti *Brannstroom/albiononline-refining-calculator*), pemain sering menderita kerugian jutaan Silver akibat kelemahan fatal pada tools yang ada saat ini:

1. **Stale Data Trap:** Tools yang ada menarik data tanpa validasi umur (*timestamp age*). Pemain melihat selisih harga tinggi, membawa modal puluhan juta Silver antar-kota, ternyata harga tersebut berasal dari order 24 jam lalu yang sudah ludes.
2. **Intermediate Tax Double-Counting (Bug Stacking Profit):** Saat pemain mengolah secara berantai (T4 -> T5 -> T6), tools yang ada memotong pajak pasar di setiap tier perantara, padahal pemain hanya membayar pajak pasar satu kali saat menjual produk akhir.
3. **Stone Unique Mechanic Discrepancy:** Komoditas Stone (Batu) tidak memiliki enchantment (`.1`, `.2`, `.3`, `.4`). Tools generik sering memaksakan schema enchant pada batu sehingga menghasilkan angka kalkulasi yang korup.
4. **Illiquidity Trap (Volume Buta):** Margin keuntungan 50% tidak ada gunanya jika volume perdagangan barang tersebut hanya 5 unit per hari di kota tujuan. Tools harus menyertakan data volume perdagangan 24 jam (`item_count` dari history endpoint AODP).
5. **Channel Pajak Kaku:** Pemain sering menjual via Discord / Guild trade (0% pajak pasar) atau pasang Buy Order (hanya 2.5% setup fee). Tools harus menyediakan kustomisasi pajak transaksi secara fleksibel (0%, 2.5%, 4%, 8%).

**Objektif:** Membangun engine berbasis web (*Astro 5 + Go 1.27 + SQLite*) yang secara khusus menyelesaikan ke-5 masalah di atas dengan akurasi 100%.

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
