// OrderBook.jsx - Displays a real-time order book with bids and asks
const OrderBook = ({ data, symbol }) => {
  if (!data || !data.bids || !data.asks) {
    return (
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4">
        <h2 className="text-xl font-semibold mb-4">Order Book</h2>
        <div className="text-center py-8 text-gray-500 dark:text-gray-400">
          No order book data available for {symbol || 'selected instrument'}
        </div>
      </div>
    );
  }

  // Calculate price precision based on price values
  const getPricePrecision = () => {
    const allPrices = [...data.bids, ...data.asks].map(level => level.price);
    if (allPrices.length === 0) return 2;
    
    const minPrice = Math.min(...allPrices);
    if (minPrice < 0.1) return 6;
    if (minPrice < 1) return 4;
    if (minPrice < 10) return 3;
    if (minPrice < 1000) return 2;
    return 1;
  };

  // Calculate volume precision
  const getVolumePrecision = () => {
    const allVolumes = [...data.bids, ...data.asks].map(level => level.volume);
    if (allVolumes.length === 0) return 4;
    
    const minVolume = Math.min(...allVolumes);
    if (minVolume < 0.001) return 6;
    if (minVolume < 0.1) return 4;
    return 2;
  };

  const pricePrecision = getPricePrecision();
  const volumePrecision = getVolumePrecision();

  // Calculate the total volume at each price level (cumulative)
  const calculateCumulativeVolumes = (levels) => {
    let cumulative = 0;
    return levels.map(level => {
      cumulative += level.volume;
      return { ...level, cumulative };
    });
  };

  const bidsWithCumulative = calculateCumulativeVolumes(data.bids);
  const asksWithCumulative = calculateCumulativeVolumes(data.asks);

  // Find max cumulative volume for scaling the depth visualization
  const maxCumulativeVolume = Math.max(
    bidsWithCumulative.length > 0 ? bidsWithCumulative[bidsWithCumulative.length - 1].cumulative : 0,
    asksWithCumulative.length > 0 ? asksWithCumulative[asksWithCumulative.length - 1].cumulative : 0
  );

  // Calculate spread
  const spread = data.asks.length > 0 && data.bids.length > 0 
    ? data.asks[0].price - data.bids[0].price 
    : 0;
  
  const spreadPercentage = data.bids.length > 0 
    ? (spread / data.bids[0].price) * 100
    : 0;

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4">
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-xl font-semibold">{symbol} Order Book</h2>
        <div className="text-sm text-gray-500 dark:text-gray-400">
          Last updated: {new Date(data.timestamp).toLocaleTimeString()}
        </div>
      </div>

      <div className="flex justify-center mb-4">
        <div className="bg-gray-100 dark:bg-gray-700 rounded-lg px-4 py-2 text-center">
          <div className="text-sm text-gray-500 dark:text-gray-400">Spread</div>
          <div className="font-bold">
            {spread.toFixed(pricePrecision)} ({spreadPercentage.toFixed(4)}%)
          </div>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        {/* Bids (Buy Orders) */}
        <div>
          <div className="flex justify-between px-2 py-1 font-semibold text-sm bg-gray-100 dark:bg-gray-700 rounded-t">
            <div className="w-1/3 text-left">Price</div>
            <div className="w-1/3 text-right">Amount</div>
            <div className="w-1/3 text-right">Total</div>
          </div>
          
          <div className="max-h-96 overflow-y-auto">
            {bidsWithCumulative.map((bid, index) => (
              <div 
                key={`bid-${index}`} 
                className="flex justify-between px-2 py-1 text-sm border-b border-gray-100 dark:border-gray-700 relative"
              >
                {/* Background bar for depth visualization */}
                <div 
                  className="absolute right-0 top-0 bottom-0 bg-green-100 dark:bg-green-900 opacity-30"
                  style={{ width: `${(bid.cumulative / maxCumulativeVolume) * 100}%` }}
                ></div>

                {/* Content */}
                <div className="w-1/3 font-mono text-green-600 dark:text-green-400 text-left z-10">
                  {bid.price.toFixed(pricePrecision)}
                </div>
                <div className="w-1/3 font-mono text-right z-10">
                  {bid.volume.toFixed(volumePrecision)}
                </div>
                <div className="w-1/3 font-mono text-gray-600 dark:text-gray-400 text-right z-10">
                  {bid.cumulative.toFixed(volumePrecision)}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Asks (Sell Orders) */}
        <div>
          <div className="flex justify-between px-2 py-1 font-semibold text-sm bg-gray-100 dark:bg-gray-700 rounded-t">
            <div className="w-1/3 text-left">Price</div>
            <div className="w-1/3 text-right">Amount</div>
            <div className="w-1/3 text-right">Total</div>
          </div>
          
          <div className="max-h-96 overflow-y-auto">
            {asksWithCumulative.map((ask, index) => (
              <div 
                key={`ask-${index}`} 
                className="flex justify-between px-2 py-1 text-sm border-b border-gray-100 dark:border-gray-700 relative"
              >
                {/* Background bar for depth visualization */}
                <div 
                  className="absolute left-0 top-0 bottom-0 bg-red-100 dark:bg-red-900 opacity-30"
                  style={{ width: `${(ask.cumulative / maxCumulativeVolume) * 100}%` }}
                ></div>

                {/* Content */}
                <div className="w-1/3 font-mono text-red-600 dark:text-red-400 text-left z-10">
                  {ask.price.toFixed(pricePrecision)}
                </div>
                <div className="w-1/3 font-mono text-right z-10">
                  {ask.volume.toFixed(volumePrecision)}
                </div>
                <div className="w-1/3 font-mono text-gray-600 dark:text-gray-400 text-right z-10">
                  {ask.cumulative.toFixed(volumePrecision)}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};
