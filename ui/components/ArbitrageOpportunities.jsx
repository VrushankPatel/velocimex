// ArbitrageOpportunities.jsx - Displays detected arbitrage opportunities
const ArbitrageOpportunities = ({ opportunities }) => {
  if (!opportunities || opportunities.length === 0) {
    return (
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4">
        <h2 className="text-xl font-semibold mb-4">Arbitrage Opportunities</h2>
        <div className="text-center py-8 text-gray-500 dark:text-gray-400">
          No arbitrage opportunities detected
        </div>
      </div>
    );
  }

  // Sort opportunities by profit percentage (descending)
  const sortedOpportunities = [...opportunities].sort((a, b) => 
    b.profitPercent - a.profitPercent
  );

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
      <div className="p-4 border-b border-gray-200 dark:border-gray-700">
        <h2 className="text-xl font-semibold">Arbitrage Opportunities</h2>
        <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
          Detected cross-exchange price differences with potential profit
        </p>
      </div>

      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead className="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                Symbol
              </th>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                Buy at
              </th>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                Sell at
              </th>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                Spread
              </th>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                Profit %
              </th>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                Est. Profit
              </th>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                Latency (ms)
              </th>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                Status
              </th>
            </tr>
          </thead>
          <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
            {sortedOpportunities.map((opportunity, index) => (
              <tr key={index} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                  {opportunity.symbol}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm">
                  <div>{opportunity.buyPrice.toFixed(2)}</div>
                  <div className="text-xs text-gray-500 dark:text-gray-400">{opportunity.buyExchange}</div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm">
                  <div>{opportunity.sellPrice.toFixed(2)}</div>
                  <div className="text-xs text-gray-500 dark:text-gray-400">{opportunity.sellExchange}</div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm">
                  {(opportunity.sellPrice - opportunity.buyPrice).toFixed(2)}
                </td>
                <td className={`px-6 py-4 whitespace-nowrap text-sm font-medium ${
                  opportunity.profitPercent >= 0.5 ? 'text-green-600 dark:text-green-400' : 
                  opportunity.profitPercent >= 0.2 ? 'text-blue-600 dark:text-blue-400' :
                  'text-gray-600 dark:text-gray-400'
                }`}>
                  {opportunity.profitPercent.toFixed(4)}%
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm">
                  ${opportunity.estimatedProfit.toFixed(2)}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm">
                  {opportunity.latencyEstimate}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm">
                  {opportunity.isValid ? (
                    <span className="px-2 py-1 text-xs rounded-full bg-green-100 dark:bg-green-800 text-green-800 dark:text-green-100">
                      Valid
                    </span>
                  ) : (
                    <span className="px-2 py-1 text-xs rounded-full bg-red-100 dark:bg-red-800 text-red-800 dark:text-red-100">
                      Invalid
                    </span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};
