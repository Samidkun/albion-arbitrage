# Panduan Pengguna AlbionFlow (End-User Guide)
## Panduan Lengkap Arbitrase Pasar & Optimalisasi Refining Albion Online

> **Versi:** 2026.09.14 · **Audience:** Albion Online Crafters, Market Tycoons, Haulers, Guild Economy Managers  
> **Aplikasi Web:** `http://localhost:4321` · **Backend Engine:** `http://localhost:8081`

---

## Daftar Isi
1. [Bab 1: Mengapa Kalkulator Generic Gagal & Menguras Jutaan Silver](#bab-1-mengapa-kalkulator-generic-gagal--menguras-jutaan-silver)
2. [Bab 2: Pengoperasian Global Control Bar](#bab-2-pengoperasian-global-control-bar)
3. [Bab 3: Masterclass Refining Optimizer & Silver Per Focus (SPF)](#bab-3-masterclass-refining-optimizer--silver-per-focus-spf)
4. [Bab 4: Audit Akurasi "Show the Math" & Ekspor Discord](#bab-4-audit-akurasi-show-the-math--ekspor-discord)
5. [Bab 5: Matriks Arbitrase Antar-Kota (/arbitrage) & Checklist Hauling](#bab-5-matriks-arbitrase-antar-kota-arbitrage--checklist-hauling)
6. [Tanya Jawab Operasional (FAQ)](#tanya-jawab-operasional-faq)

---

## Bab 1: Mengapa Kalkulator Generic Gagal & Menguras Jutaan Silver

Banyak pemain Albion Online mengeluhkan kerugian jutaan Silver meskipun telah mengikuti angka di kalkulator atau spreadsheet publik. Kerugian tersebut hampir selalu disebabkan oleh **4 jebakan sistemik** yang diabaikan oleh tools generic:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       4 JEBAKAN FATAL KALKULATOR LAMA                      │
├──────────────────────────┬──────────────────────────────────────────────────┤
│ 1. Stale Data Trap       │ Mengambil harga dari order 24 jam lalu yang sudah│
│    (Data Basi >60 Menit) │ habis; pemain memborong kargo bernilai semu.     │
├──────────────────────────┼──────────────────────────────────────────────────┤
│ 2. Intermediate Tax      │ Memotong pajak pasar di setiap tier perantara    │
│    Double-Counting       │ (T4->T5->T6), memotong estimasi profit hingga 40%│
├──────────────────────────┼──────────────────────────────────────────────────┤
│ 3. Illiquidity Trap      │ Menjanjikan margin 50% pada barang yang hanya    │
│    (Volume Buta)         │ terjual 2 unit/hari; modal macet berminggu-minggu│
├──────────────────────────┼──────────────────────────────────────────────────┤
│ 4. Rogue Station Owners  │ Kartel pemilik toko tempa menaikkan pajak ke     │
│    (Kartel Toko Tempa)   │ 9,999 silver/nutrisi semalam tanpa terdeteksi.   │
└──────────────────────────┴──────────────────────────────────────────────────┘
```

### 1.1 Stale Price Trap (Jebakan Harga Basi > 60 Menit)
Kalkulator biasa mengambil harga dari The Albion Online Data Project (AODP) tanpa memverifikasi kapan transaksi tersebut tercatat. Seringkali harga murah suatu komoditas tercatat 18 jam lalu dan stoknya sudah diborong pemain lain. Saat Anda tiba di kota tujuan, harga beli riil sudah jauh lebih tinggi.
* **Solusi AlbionFlow:** Mesin kami menerapkan *strict 60-minute freshness guard*. Setiap harga memiliki status kesegaran (Fresh <15m, Aging 15-60m, Stale >60m). Order yang lebih tua dari 60 menit otomatis disaring dari rekomendasi utama.

### 1.2 Intermediate Tax Double-Counting (Pajak Berantai Ganda)
Jika Anda mengolah kayu dari T4 menjadi T5 lalu T6 (Chained Refining), Anda **tidak menjual** papan T4 atau T5 Anda ke pasar; Anda menyimpannya di kantong dan menggunakannya sebagai bahan baku tier berikutnya.
* Kalkulator lama memperlakukan setiap tier seolah-olah dijual dan dibeli kembali di pasar, memotong pajak 4%-8% dan biaya registrasi order 2.5% berkali-kali.
* **Solusi AlbionFlow:** Opsi **Chained Refining** menghitung modal riil berantai secara rekursif dari harga bahan mentah dasar tanpa pemotongan pajak perantara fiktif.

### 1.3 Illiquidity Trap (Jebakan Barang Tidak Laku)
Menemukan rute arbitrase dengan margin keuntungan 70% di kertas terasa hebat, sampai Anda menyadari bahwa barang tersebut hanya laku 3 unit per 24 jam di kota tujuan. Membawa 500 unit ke sana akan membekukan modal puluhan juta Silver Anda selama berbulan-bulan.
* **Solusi AlbionFlow:** Setiap baris dilengkapi data volume perdagangan riil 24 jam (`traded_volume_24h`). Barang dengan volume < 5 unit/hari otomatis diberi tanda merah `ILLIQUID` dan dapat disaring dengan satu klik.

### 1.4 Rogue Station Owners (Pajak Toko Tempa Liar)
Pemilik crafting station di Royal Cities seringkali mengubah pajak (*Usage Fee*) dari 500 Silver menjadi 9,999 Silver per 100 Nutrisi secara mendadak untuk memeras pemain yang tidak memperhatikan layar konfirmasi.
* **Solusi AlbionFlow:** Slider **Station Tax** di Control Bar memungkinkan Anda mengunci batas pajak aktual yang Anda temukan di kota secara live, seketika memperbarui seluruh kalkulasi laba bersih.

---

## Bab 2: Pengoperasian Global Control Bar

Control Bar di bagian atas dashboard adalah pusat komando ekonomi Anda. Setiap perubahan parameter akan langsung memicu penghitungan ulang secara instan (*zero-latency recalculation*).

![Control Bar Overview](https://raw.githubusercontent.com/placeholder/control-bar-preview.png)

### 2.1 Pemilih Wilayah Server (Region Selector)
Pilih server tempat karakter Anda aktif di pojok kanan atas:
- **Asia (Singapore):** Server Asia Timur & Tenggara.
- **Americas (Washington D.C.):** Server Benua Amerika.
- **Europe (Amsterdam):** Server Eropa & Timur Tengah.

*Catatan: Seluruh data harga, order volume, dan timestamp diisolasi secara ketat per wilayah untuk mencegah kontaminasi data lintas server.*

### 2.2 Filter Komoditas & Tier
- **Commodity Filter:** Pilih antara `All`, `Wood` (Fort Sterling), `Ore` (Thetford), `Fiber` (Lymhurst), `Hide` (Martlock), atau `Stone` (Bridgewatch). Setiap tombol menampilkan lencana kota dengan spesialisasi bonus pengembalian bahan.
- **Tier Filter:** Filter cepat untuk `All`, `T4`, `T5`, `T6`, `T7`, atau `T8`.

### 2.3 Pengatur Pajak Toko Tempa (Station Tax Slider)
- **Rentang Nilai:** 200 hingga 2,000 Silver per 100 Nutrition (default: 600 Silver).
- **Cara Penggunaan:** Cek station termurah di kota tempat Anda akan mengolah bahan, lalu geser slider ke angka tersebut. AlbionFlow akan menghitung ulang biaya produksi per unit seketika.

### 2.4 Saluran Pajak Penjualan (Sales Tax Channels)
AlbionFlow mendukung 3 saluran penjualan nyata dalam game:
1. **Direct Trade (Pajak 0%):**
   - Digunakan jika Anda menjual langsung lewat jendela trade antar-pemain, kontrak guild, atau saluran Discord trader.
   - Tanpa potongan pajak pasar (*Sales Tax 0%*) dan tanpa biaya pasang order (*Setup Fee 0%*).
2. **Instant Buy Order (Pajak 4% Premium / 8% Non-Premium):**
   - Digunakan jika Anda langsung mencairkan barang ke Buy Order yang sudah ada di pasar (*Sell to Order*).
   - Tanpa biaya pasang order (*Setup Fee 0%*), hanya dipotong pajak penjualan pasar.
3. **Sell Order (Pajak 4%/8% + Setup Fee 2.5%):**
   - Saluran standar di mana Anda memasang listing jual di pasar (*Create Sell Order*).
   - Dikenakan biaya pendaftaran order 2.5% plus pajak pasar saat barang laku (total potongan 6.5% untuk Premium, 10.5% untuk Non-Premium).

### 2.5 Sakelar Chained Refining
- **OFF (Beli Bahan Refined Sebelumnya):** Model standar. Asumsikan Anda membeli batangan/papan refined tier sebelumnya langsung dari pasar lokal.
- **ON (Refining Berantai Rekursif):** Model master crafter. Menghitung biaya pengolahan mandiri dari tier paling bawah (T4 mentah) secara bertingkat dengan efisiensi RRR penuh tanpa potongan pajak pasar ganda.

---

## Bab 3: Masterclass Refining Optimizer & Silver Per Focus (SPF)

Refining adalah salah satu cara paling stabil untuk mencetak puluhan juta Silver pasif setiap hari di Albion Online jika Anda memahami metrik **Silver Per Focus (SPF)** dan **Resource Return Rate (RRR)**.

### 3.1 Resource Return Rate (RRR) Resmi Game
Ketika Anda mengolah bahan di Albion, persentase tertentu dari bahan baku akan dikembalikan langsung ke inventory Anda:

$$\text{Jumlah Bahan Bersih Terpakai} = \text{Bahan Dasar} \times (1 - \text{RRR})$$

| Lokasi & Kondisi | RRR Standar | Efek Pengembalian Bahan |
|---|---|---|
| **Kota Biasa / Pulau Pribadi (Tanpa Bonus Biome)** | **15.2%** | Membayar 84.8% bahan dasar |
| **Kota dengan Spesialisasi Biome Resmi** | **36.7%** | Membayar 63.3% bahan dasar (Hemat ~36.7%) |
| **Kota Spesialisasi Biome + Crafting Focus** | **53.9%** | Membayar 46.1% bahan dasar (Hemat >setengah bahan!) |

#### Peta Kota Spesialisasi Biome Resmi:
- **Fort Sterling:** Wood (Kayu / Planks)
- **Thetford:** Ore (Biji Besi / Bars)
- **Lymhurst:** Fiber (Serat Tanaman / Cloth)
- **Bridgewatch:** Stone (Batu / Blocks)
- **Martlock:** Hide (Kulit Hewan / Leather)

> **PENTING — Pengecualian Khusus Komoditas Stone (Batu):**  
> Di Albion Online, Stone Block **tidak memiliki enchantment** (.1, .2, .3, .4). Jangan pernah mencari atau membeli batu ber-enchant. Sistem AlbionFlow secara otomatis menonaktifkan filter enchantment untuk Stone agar kalkulasi tidak menghasilkan data fiktif.

### 3.2 Memahami Metrik Silver Per Focus (SPF)
Setiap karakter Premium menerima 10,000 Focus Points setiap hari (maksimal akumulasi 30,000). Jangan pernah membakar Focus secara acak!

$$\text{SPF} = \frac{\text{Profit Bersih (Dengan Focus)} - \text{Profit Bersih (Tanpa Focus)}}{\text{Biaya Focus Aktual Per Unit}}$$

#### Tabel Evaluasi Kualitas SPF:
| Nilai SPF | Kategori | Rekomendasi Tindakan |
|---|---|---|
| **> 100 Silver / Focus** | **Tier S (Excellent)** | Prioritas utama! Eksekusi segera seluruh 10,000 Focus harian Anda di sini (>1,000,000 Silver profit murni per hari). |
| **50 – 100 Silver / Focus** | **Tier A (Good)** | Sangat layak. Pilihan stabil untuk komoditas volume tinggi (T6.1, T7.1). |
| **20 – 50 Silver / Focus** | **Tier B (Fair)** | Cukup baik jika Anda sedang menaikkan level spesialisasi (fame grinding). |
| **< 20 Silver / Focus** | **Tier C (Poor)** | **HINDARI!** Menggunakan Focus di sini membuang potensi keuntungan. Lebih baik simpan Focus Anda. |

### 3.3 Pengaruh Mastery Node (0–100)
Tingkat spesialisasi Anda di Destiny Board mengurangi konsumsi Focus secara dramatis:
- Pada **Mastery 0**, biaya Focus adalah 100% dari biaya dasar.
- Pada **Mastery 100**, konsumsi Focus terpangkas hingga **50%** dari biaya dasar.
- Pengurangan Focus ini menggandakan nilai SPF Anda secara efektif, melipatgandakan yield harian dari 10,000 Focus Anda.

### 3.4 Indikator Kesegaran Data (Data Freshness Badges)
Perhatikan lingkaran status di kolom **DATA AGE**:
- 🟢 **Hijau (Fresh < 15 menit):** Data harga sangat segar dari pemain yang baru membuka pasar dengan AODP client aktif. Eksekusi aman.
- 🟡 **Amber (Aging 15–60 menit):** Data mulai menua. Disarankan untuk mengecek ketersediaan lot pasar di game sebelum membeli modal besar.
- 🔴 **Merah (Stale > 60 menit):** Data berumur lebih dari 1 jam. Kemungkinan harga sudah bergerak atau lot murah telah dibeli.

---

## Bab 4: Audit Akurasi "Show the Math" & Ekspor Discord

AlbionFlow menganut prinsip transparansi total (*Anti-Black-Box*). Kami tidak hanya memberi Anda angka profit; kami memperlihatkan seluruh rincian matematika di baliknya.

### 4.1 Mengakses Modal Audit
Klik tombol **`[Show the Math]`** atau **`[Rincian]`** pada baris mana pun di Refining Matrix untuk membuka laci audit step-by-step.

```
┌────────────────────────────────────────────────────────────┐
│ [T6.1] Bloodoak Planks  •  Fort Sterling  •  36.7% RRR     │
├────────────────────────────────────────────────────────────┤
│ 01. BAHAN MENTAH (RAW MATERIAL)                            │
│     Kebutuhan Dasar: 4x T6 Cedar Logs                      │
│     Harga Beli: 420 Silver x 4 = 1,680 Silver              │
│     Pengembalian RRR (36.7%): -616.56 Silver               │
│     -> Biaya Bersih Bahan Mentah: 1,063.44 Silver          │
├────────────────────────────────────────────────────────────┤
│ 02. BAHAN REFINED SEBELUMNYA (PREVIOUS TIER REFINED)       │
│     Kebutuhan Dasar: 1x T5 Pine Planks                     │
│     Harga Beli: 510 Silver x 1 = 510 Silver                │
│     Pengembalian RRR (36.7%): -187.17 Silver               │
│     -> Biaya Bersih Refined Sebelumnya: 322.83 Silver      │
├────────────────────────────────────────────────────────────┤
│ 03. ONGKOS SEWA TOKO TEMPA (STATION USAGE FEE)             │
│     Item Value: 256  •  Tarif Pajak: 600 Silver/100 Nut    │
│     -> Biaya Nutrisi Toko: 172.80 Silver                   │
├────────────────────────────────────────────────────────────┤
│ 04. PENDAPATAN PENJUALAN & PAJAK (GROSS REVENUE & TAXES)   │
│     Harga Jual Pasar: 2,450 Silver                         │
│     Sales Tax (4% Premium): -98.00 Silver                  │
│     Market Setup Fee (2.5%): -61.25 Silver                 │
│     -> Pendapatan Bersih (Net Revenue): 2,290.75 Silver    │
├────────────────────────────────────────────────────────────┤
│ 05. HASIL AKHIR (NET RETURN & EFFICIENCY)                  │
│     Total Modal Produksi: 1,559.07 Silver                  │
│     Keuntungan Bersih (Per Unit): +731.68 Silver           │
│     Profit Margin (ROI): +46.93%                           │
│     Silver Per Focus: 94.20 Silver / Focus                 │
└────────────────────────────────────────────────────────────┘
```

### 4.2 Fitur Ekspor Satu-Klik ke Discord
Klik tombol **`[Salin Format Discord]`** di bagian bawah modal audit untuk menyalin ringkasan terformat yang siap dibagikan ke guild chat, Discord trade channel, atau catatan pribadi:

```markdown
**[AlbionFlow Audit] T6.1 Bloodoak Planks**
- Lokasi: Fort Sterling (Bonus RRR: 36.7%)
- Biaya Bahan Mentah (Net): 1,063 Silver
- Biaya Refined T5 (Net): 323 Silver
- Ongkos Toko (600/100 Nut): 173 Silver
- Total Modal Produksi: 1,559 Silver
- Harga Jual Kotor: 2,450 Silver
- Pajak Pasar & Setup (6.5%): -159 Silver
- **Profit Bersih / Unit:** +732 Silver (+46.9% Margin)
- **Efisiensi Focus (SPF):** 94.20 Silver/Focus (Daily Yield: 942k Silver)
*Dihitung via AlbionFlow Engine (Data Age: 8m ago)*
```

---

## Bab 5: Matriks Arbitrase Antar-Kota (`/arbitrage`) & Checklist Hauling

Fitur arbitrase antar-kota (*Cross-City Hauling Matrix*) memindai selisih harga komoditas antar 6 Royal Cities (Thetford, Fort Sterling, Lymhurst, Bridgewatch, Martlock, Caerleon) dan Black Market.

### 5.1 Menavigasi Matriks Rute Arbitrase
Akses tab **`/arbitrage`** pada navigasi atas:
- **Origin City (Kota Asal):** Kota tempat Anda memborong barang dengan harga murah.
- **Target City (Kota Tujuan):** Kota tempat Anda menjual kargo dengan harga tinggi.
- **Tombol Swap (⇄):** Tukar kota asal dan tujuan secara instan untuk memeriksa rute pulang-pergi (*backhaul trade*).
- **Min Net Profit & Min Margin:** Saring rute yang memberi laba di atas batas minimal yang Anda inginkan (misal: minimal 500 Silver/unit dan minimal 15% margin).

### 5.2 Membaca Lencana Likuiditas & Risiko
Setiap baris rute dilengkapi dengan indikator keselamatan:

```
[Fort Sterling -> Caerleon]  T6.1 Wood  •  Spread: +1,240 Silver  [HIGH RISK]  [LIQUID: 412/day]
[Thetford -> Lymhurst]       T5.0 Ore   •  Spread: +410 Silver    [LOW RISK]   [LIQUID: 1,820/day]
[Martlock -> Bridgewatch]    T8.3 Hide  •  Spread: +18,500 Silver [LOW RISK]   [ILLIQUID: 2/day]
```

1. **Lencana Likuiditas Pasar (24h Volume):**
   - 🟢 **HIGH (> 100 unit/hari):** Permintaan pasar sangat tinggi. Barang Anda dijamin terjual cepat tanpa menjatuhkan harga pasar.
   - 🟡 **MED (20–100 unit/hari):** Pasar moderat. Aman untuk volume angkut sedang (50–100 unit per trip).
   - 🔴 **LOW / ILLIQUID (< 20 unit/hari):** **Waspada!** Margin mungkin tinggi tetapi order beli sangat sedikit. Risiko modal macet.

2. **Lencana Tingkat Risiko (Risk Tier Badges):**
   - 🟢 **LOW RISK (Rute Biru / Kuning):** Perjalanan antar 5 Royal Cities utama (Thetford, Martlock, Fort Sterling, Lymhurst, Bridgewatch). Rute ini berada di Safe Zone (Blue/Yellow Zone). Anda tidak bisa di-loot oleh pemain lain jika jatuh.
   - 🟠 **MED RISK (Rute Panjang / Margin Tipis):** Melintasi beberapa map dengan margin keuntungan yang rentan tergerus fluktuasi harga cepat.
   - 🔴 **HIGH RISK (Rute Caerleon / Red Zone):** Rute yang masuk atau keluar dari Caerleon melintasi **Red Zone**. Pemain lain dapat membunuh Anda dan menjarah seluruh kargo Anda (*Full Loot PvP*). Wajib menggunakan gear anti-gank atau pengawalan guild!

### 5.3 Checklist Wajib Sebelum Berangkat Hauling
Sebelum Anda memborong barang di kota asal, jalankan checklist 5 langkah ini:

- [ ] **1. Cek Kesegaran Data:** Pastikan rute bertanda status Hijau (<15 menit) atau cek fisik order pasar di kota asal secara langsung.
- [ ] **2. Cek Volume Likuiditas Tujuan:** Pastikan volume harian di kota tujuan minimal 3x lebih besar daripada jumlah barang yang Anda angkut.
- [ ] **3. Perhitungan Berat & Mount:**
  - Pastikan kapasitas angkut mount Anda mencukupi (contoh: *Transport Ox* untuk zona aman, *Grizzly Bear* atau *Spectral Direboar* untuk kapasitas berat plus pertahanan).
  - Berat inventory tidak boleh melebihi 100% agar mount tidak melambat!
- [ ] **4. Jalur Aman vs Red Zone:**
  - Jika menuju Caerleon, periksa jumlah hostile player di setiap perbatasan map (angka merah di bawah minimap).
  - Gunakan *Invisibility Potion* dan skill mobilitas (Miner Workboots, Double Bladed Staff / Bloodletter).
- [ ] **5. Pajak Pasar & Setup Fee:**
  - Pastikan Anda memperhitungkan biaya pasang listing (2.5%) dan pajak pasar (4% Premium) saat menentukan harga jual akhir.

---

## Tanya Jawab Operasional (FAQ)

### Q: Apakah menggunakan AlbionFlow melanggar aturan game (ToS Sandbox Interactive)?
**A: 100% Legal dan Aman.** AlbionFlow tidak membaca RAM komputer Anda (*zero memory injection*) dan tidak mengendus paket internet game (*zero packet sniffing*). Seluruh data ditarik secara legal melalui REST API publik *The Albion Online Data Project* (AODP) yang dikontribusikan secara sukarela oleh komunitas Albion.

### Q: Mengapa harga di game berbeda sedikit dengan yang ada di dashboard?
**A:** Pasar Albion Online sangat dinamis. Jika pemain lain baru saja membeli habis lot termurah beberapa detik lalu, harga dasar akan naik ke tier berikutnya. Gunakan indikator kesegaran data (< 15 menit) untuk meminimalisir deviasi harga.

### Q: Saya tidak memiliki status Premium, apakah kalkulasi tetap akurat?
**A:** Ya! Anda cukup mematikan sakelar **PREMIUM** di Control Bar. AlbionFlow akan otomatis menyesuaikan pajak penjualan dari 4% menjadi 8% dan menghitung ulang seluruh margin profit.

### Q: Apakah saya harus mengolah bahan di kota bonusnya?
**A:** Sangat disarankan! Mengolah di kota dengan spesialisasi biome memberikan pengembalian bahan 36.7% (vs 15.2% di kota non-bonus). Selisih 21.5% pengembalian bahan ini seringkali menjadi penentu mutlak antara untung ratusan ribu Silver atau rugi total.
