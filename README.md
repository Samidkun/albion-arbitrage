# ⚔️ Albion Arbitrage — Cross-City Market Arbitrage Scanner

> **Real-time Albion Online cross-city trading and transport arbitrage engine built with Go, Astro 5, and SQLite.**

---

## 📸 Visual Showcase & Market Telemetry

<p align="center">
  <img src="docs/screenshots/preview.png" alt="Market Overview" width="100%" style="border-radius: 10px; border: 1px solid rgba(255,255,255,0.1);" />
</p>
<p align="center"><em>Figure 1: Royal Continent Cross-City Market Arbitrage Dashboard (Bridgewatch, Fort Sterling, Lymhurst, Martlock, Thetford, Caerleon).</em></p>

<br />

<div align="center">
  <table width="100%">
    <tr>
      <td width="50%" align="center">
        <img src="docs/screenshots/01-market-overview.png" alt="Market Overview" width="100%" style="border-radius: 8px;" />
        <br /><strong>Figure 2: Real-time Price Tickers</strong><br />
        <em>Live AODP market order ingestion and cross-city price spread calculations.</em>
      </td>
      <td width="50%" align="center">
        <img src="docs/screenshots/02-arbitrage-scanner.png" alt="Arbitrage Scanner" width="100%" style="border-radius: 8px;" />
        <br /><strong>Figure 3: Profit Margin & Risk Scanner</strong><br />
        <em>Net profit margin ranking after deducting market taxes, setup fees, and transport weight capacities.</em>
      </td>
    </tr>
  </table>
</div>

---

## 🏗️ Architecture

- **Go Backend Service:** High-concurrency market parser pulling live orders from the Albion Online Data Project (AODP).
- **Astro 5 Frontend:** Zero-JS baseline with island architecture for ultra-responsive search and sorting.
- **SQLite In-Memory Cache:** Fast spatial queries for trade route optimization.
