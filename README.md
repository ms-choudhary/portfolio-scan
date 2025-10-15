# portfolio-scan

Tracks your current mutual fund portfolio for different asset classes. And helps you rebalance if the portfolio gets skewed from the required allocation.

![screenshot.png](:/docs/screenshot.png)

On every reload it fetches the current NAV of all funds from AMFI site. 

There are two ways to add funds to your portfolio:
- Create a `.portfolio` file with class, name and quantity. Current price will be fetched automatically from AMFI site. 
- Fetch funds automatically from zerodha, see below

## Installation

- Build UI
```
$ cd ui
$ npm install
$ npm run build
```

- Start app
```
$ go run main.go
```

## Fetch funds from Zerodha

- Create a account on: https://developers.kite.trade/apps
- Create a new app, requires your zerodha client id, which you can obtain from kite account. 
- Keep a note of API Key & API Secret. 
- Export envs
```
export EQ_KITE_API_KEY=<dummy>
export EQ_KITE_API_SECRET=<dummy>
export DEBT_KITE_API_KEY=<dummy>
export DEBT_KITE_API_SECRET=<dummy>
```

### Limitation

- Each app can only fetch funds from single zerodha account, if you've separate accounts for funds, you need to create different apps for each of them. For eg, I keep my equity and debt in to separate accounts. 
- Even though zerodha tracks the current price of holdings, it lags behind that what's available at AMFI. 
- You cannot renew tokens easily, so essentially your session is just limited to a single day. However, you don't need to login again after `.portfolio` is generated. Any time you want to refresh your holdings, after purchasing new funds, you can delete `.portfolio` and refetch the funds again. 
