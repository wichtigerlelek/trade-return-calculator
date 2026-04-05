# Trade Return Calculator

A tiny trade simulator written in Go that visualizes the long-term outcome of a trading strategy across hundreds or thousands of simulated trades. It generates interactive charts showing how different win rates, risk levels, and risk-reward ratios affect your balance over time.

## Screenshots

### CLI Setup

<img width="589" height="290" alt="Screenshot from 2026-04-05 23-02-51" src="https://github.com/user-attachments/assets/797094f4-bb50-45f0-a21a-6abfd3d57604" />


### Percentile Cloud Chart

<img width="1143" height="720" alt="Screenshot from 2026-04-05 23-03-18" src="https://github.com/user-attachments/assets/83ad6b51-6742-4f3a-b630-9dfffeb2bcd4" />


### Monte Carlo Lines Chart

<img width="1143" height="720" alt="Screenshot from 2026-04-05 23-03-45" src="https://github.com/user-attachments/assets/60995c8c-5a49-4839-a96a-0ed8e98a4753" />


---

## Installation

**Prerequisites:** Go 1.21+

```bash
git clone https://github.com/wichtigerlelek/trade-return-calculator.git
cd trade-return-calculator
go mod tidy
go run .
```

---

## Usage

Run the program and you will be presented with a terminal UI to configure the simulation:

```
╔══════════════════════════════════╗
║       Trade Simulator Setup      ║
╚══════════════════════════════════╝

  Use ↑↓/Tab to navigate, Enter to run

▶ Start Balance:     10000
  Number of Trades:  2000
  Simulations:       25
  Typical Risk:      0.05
  Risk-Reward:       2.0
  Win Rate:          0.38
```

| Field | Description | Default |
|---|---|---|
| Start Balance | Initial account balance | `10000` |
| Number of Trades | How many trades to simulate per run | `2000` |
| Simulations | Number of independent Monte Carlo paths | `25` |
| Typical Risk | Fraction of balance risked per trade | `0.05` (5%) |
| Risk-Reward | Reward multiplier relative to risk | `2.0` (1:2) |
| Win Rate | Probability of winning each trade | `0.38` (38%) |

- Press **Enter** on any field to run the simulation with the current values
- Leave a field blank and press **Enter** to use the default value
- Use **↑↓** to navigate between fields

After running, `chart.html` is generated and automatically opened in your default browser.

---

## How It Works

Each trade is simulated as follows:

- **Win:** `balance += balance × risk × riskReward`
- **Loss:** `balance -= balance × risk`

It also shows the theoretical break even win rate (the winrate depending on risk and riskReward at which no money is gained or lost on average) calculated with the following formular:

```
breakEvenWinRate = log(1 - risk) / (log(1 - risk) - log(1 + risk × riskReward))
```
### Sample for a Risk Reward of 1:1
| Risk per trade | Break-even win rate |
|---|---|
| 1% | ~50.25% |
| 5% | ~51.22% |
| 10% | ~52.38% |
| 25% | ~55.56% |

---

## Dependencies

| Package | Purpose |
|---|---|
| [`go-echarts`](https://github.com/go-echarts/go-echarts) | Interactive chart rendering |
| [`bubbletea`](https://github.com/charmbracelet/bubbletea) | Terminal UI framework |
