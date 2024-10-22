const apiUrl = 'http://localhost:8080/market'; // HTTP proxy endpoint for gRPC requests

let chart;
let tradingPair = 'ADA/USDT'; // Default trading pair

const tradingPairs = {
    binance: ['BTC/USDT', 'ETH/USDT', 'LTC/USDT', 'XRP/USDT', 'BCH/USDT', 'EOS/USDT', 'XLM/USDT', 'ADA/USDT', 'TRX/USDT', 'BNB/USDT', 'XMR/USDT', 'DASH/USDT'],
    kucoin: ['BTC/USDT', 'ETH/USDT', 'LTC/USDT', 'XRP/USDT', 'BCH/USDT', 'EOS/USDT', 'XLM/USDT', 'ADA/USDT', 'TRX/USDT', 'BNB/USDT', 'XMR/USDT', 'DASH/USDT']
};

// Populate the dropdown with trading pairs
function populateDropdown() {
    const selectElement = document.getElementById('tradingPairSelect');
    selectElement.innerHTML = ''; // Clear previous options

    // Get unique trading pairs across exchanges
    const uniquePairs = [...new Set([...tradingPairs.binance, ...tradingPairs.kucoin])];

    uniquePairs.forEach(pair => {
        const option = document.createElement('option');
        option.value = pair;
        option.textContent = pair;
        selectElement.appendChild(option);
    });

    // Set the default value
    selectElement.value = tradingPair;

    // Add an event listener to update the chart when a new pair is selected
    selectElement.addEventListener('change', (event) => {
        tradingPair = event.target.value;
        fetchMarketData(); // Fetch and update the chart with the new trading pair
    });
}

function fetchMarketData() {
    fetch(apiUrl, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ trading_pair: tradingPair }), // Use the selected trading pair
    })
        .then(response => response.json())
        .then(data => {
            updateMarketTable(data.markets);
            updateChart(data.markets);
        })
        .catch(error => {
            console.error('Error fetching market data:', error);
        });
}

function updateMarketTable(markets) {
    const marketDataElement = document.getElementById('market-data');
    marketDataElement.innerHTML = ''; // Clear previous data

    markets.forEach(market => {
        const row = document.createElement('tr');

        // Create table cells
        const exchangeCell = document.createElement('td');
        const lastTradedPriceCell = document.createElement('td');
        const bestBuyPriceCell = document.createElement('td');
        const bestSellPriceCell = document.createElement('td');
        const volumeCell = document.createElement('td');
        const timestampCell = document.createElement('td');

        // Set the cell values
        exchangeCell.textContent = market.exchange;
        lastTradedPriceCell.textContent = market.last_traded_price;
        bestBuyPriceCell.textContent = market.best_buy_price;
        bestSellPriceCell.textContent = market.best_sell_price;
        volumeCell.textContent = market.volume_24hr;

        // Convert the timestamp to a human-readable format
        const timestamp = new Date(parseInt(market.timestamp.seconds) * 1000);
        timestampCell.textContent = timestamp.toLocaleString();

        // Append the cells to the row
        row.appendChild(exchangeCell);
        row.appendChild(lastTradedPriceCell);
        row.appendChild(bestBuyPriceCell);
        row.appendChild(bestSellPriceCell);
        row.appendChild(volumeCell);
        row.appendChild(timestampCell);

        // Append the row to the table body
        marketDataElement.appendChild(row);
    });
}

function updateChart(markets) {
    const exchanges = markets.map(market => market.exchange);
    const lastTradedPrices = markets.map(market => parseFloat(market.last_traded_price));
    const bestBuyPrices = markets.map(market => parseFloat(market.best_buy_price));
    const bestSellPrices = markets.map(market => parseFloat(market.best_sell_price));

    // Calculate the min and max across all price values to make the scales closer
    const allPrices = [...lastTradedPrices, ...bestBuyPrices, ...bestSellPrices];
    const minPrice = Math.min(...allPrices);
    const maxPrice = Math.max(...allPrices);

    if (!chart) {
        // Initialize the chart only once
        const ctx = document.getElementById('priceChart').getContext('2d');
        chart = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: exchanges,
                datasets: [
                    {
                        label: 'Last Traded Price',
                        data: lastTradedPrices,
                        backgroundColor: 'rgba(75, 192, 192, 0.2)',
                        borderColor: 'rgba(75, 192, 192, 1)',
                        borderWidth: 1
                    },
                    {
                        label: 'Best Buy Price',
                        data: bestBuyPrices,
                        backgroundColor: 'rgba(54, 162, 235, 0.2)',
                        borderColor: 'rgba(54, 162, 235, 1)',
                        borderWidth: 1
                    },
                    {
                        label: 'Best Sell Price',
                        data: bestSellPrices,
                        backgroundColor: 'rgba(255, 99, 132, 0.2)',
                        borderColor: 'rgba(255, 99, 132, 1)',
                        borderWidth: 1
                    }
                ]
            },
            options: {
                scales: {
                    y: {
                        beginAtZero: false, // Do not start at 0
                        min: minPrice * 0.98, // Slightly lower than the minimum price
                        max: maxPrice * 1.02, // Slightly higher than the maximum price
                        ticks: {
                            // Adjust the step size based on the price range
                            stepSize: (maxPrice - minPrice) / 5
                        }
                    }
                }
            }
        });
    } else {
        // Update the chart data instead of recreating the chart
        chart.data.labels = exchanges;
        chart.data.datasets[0].data = lastTradedPrices;
        chart.data.datasets[1].data = bestBuyPrices;
        chart.data.datasets[2].data = bestSellPrices;

        // Dynamically update the min and max values of the y-axis
        chart.options.scales.y.min = minPrice * 0.98;
        chart.options.scales.y.max = maxPrice * 1.02;
        chart.update();
    }
}


// Fetch market data every 10 seconds
setInterval(fetchMarketData, 1000);

// Fetch data immediately on page load and populate the dropdown
populateDropdown();
fetchMarketData();
