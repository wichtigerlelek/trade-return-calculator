# Trade Return Calculator

A tiny trade simulator written in Go that visualizes the long-term outcome of a trading strategy across hundreds or thousands of simulated trades. It generates interactive charts showing how different win rates, risk levels, and risk-reward ratios affect your balance over time.

## Screenshots

### Percentile Cloud Chart

<img width="1118" height="724" alt="Screenshot from 2026-04-06 14-27-42" src="https://github.com/user-attachments/assets/6561c0f0-5179-40d6-9a9a-29813360ff07" />

### Monte Carlo Lines Chart

<img width="1118" height="724" alt="Screenshot from 2026-04-06 14-27-55" src="https://github.com/user-attachments/assets/5bebd7d5-cc2b-4a34-945f-c7ac39cf71cb" />

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
╔═════════════════════════════════╗
║     TRADE RETURN CALCULATOR     ║
╚═════════════════════════════════╝

  Start Balance:         $10000
  Number of Trades:      2000
  Trades per year:       200
  Simulations:           20
  Risk Per Trade:        0.02
  Risk-Reward:           1:1.0
  Win Rate:              0.55
  Max $ Risk Cap:        $50000
▶ Slippage:              0.005

  Press Enter on last field to run

=== Configuration ===
Start Balance:      $10000.00
Number of Trades:   2000
Trades / year:      200
Simulations:        20
Typical Risk:       0.05
Risk-Reward:        1:1.0
Win Rate:           0.5800
Max Risk:           $50000.00
Slippage:           0.005

=== Analytics ===
Break-Even WR:      0.5125
Median Ending:      $10808981.83
Total Net P/L:      $223889704.36
Avg Max Drawdown:   51.16%
Worst Drawdown:     70.69%
Per-Trade Sortino:  0.1930
Annualized Sortino: 2.7293
```

| Field | Description | Default |
|---|---|---|
| Start Balance | Initial account balance | `10000` |
| Number of Trades | How many trades to simulate per run | `2000` |
| Simulations | Number of independent Monte Carlo paths | `20` |
| Typical Risk | Fraction of balance risked per trade | `0.02` (2%) |
| Risk-Reward | Reward multiplier relative to risk | `1.0` (1:1) |
| Win Rate | Probability of winning each trade | `0.55` (55%) |
| Max $ Risk Cap | Maximum amount of money a trade can be | `50000` |
| Slippage | Money lost through Volatility, Volume, Liquidity, ... | `0.005` (0.5%) |

- Press **Enter** on any field to run the simulation with the current values
- Leave a field blank and press **Enter** to use the default value
- Use **↑↓** to navigate between fields

After running, `chart.html` is generated and automatically opened in your default browser.

---

## How It Works

Each trade is simulated as follows:

- **Win:** `balance += (riskAmount * (rr - slippage))`
- **Loss:** `balance -= (riskAmount * (1 + slippage))`

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
