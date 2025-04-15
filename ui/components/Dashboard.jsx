// Dashboard.jsx - Main dashboard showing system status and strategy performance
const Dashboard = ({ strategies, connected }) => {
  const chartRef = React.useRef(null);
  const [chartInstance, setChartInstance] = React.useState(null);

  // Set up performance chart when strategies change
  React.useEffect(() => {
    if (strategies.length > 0 && chartRef.current) {
      // Destroy existing chart if it exists
      if (chartInstance) {
        chartInstance.destroy();
      }

      // Create performance chart data
      const strategyNames = strategies.map(strategy => strategy.name || 'Unknown');
      const profitLossData = strategies.map(strategy => strategy.profitLoss || 0);
      const winRates = strategies.map(strategy => {
        if (strategy.metrics && strategy.metrics.winRate) {
          return strategy.metrics.winRate * 100;
        }
        return 0;
      });

      // Create chart
      const ctx = chartRef.current.getContext('2d');
      const newChartInstance = new Chart(ctx, {
        type: 'bar',
        data: {
          labels: strategyNames,
          datasets: [
            {
              label: 'Profit/Loss',
              data: profitLossData,
              backgroundColor: 'rgba(59, 130, 246, 0.5)',
              borderColor: 'rgb(59, 130, 246)',
              borderWidth: 1
            },
            {
              label: 'Win Rate %',
              data: winRates,
              backgroundColor: 'rgba(16, 185, 129, 0.5)',
              borderColor: 'rgb(16, 185, 129)',
              borderWidth: 1,
              yAxisID: 'y1'
            }
          ]
        },
        options: {
          responsive: true,
          scales: {
            y: {
              beginAtZero: true,
              title: {
                display: true,
                text: 'Profit/Loss'
              }
            },
            y1: {
              beginAtZero: true,
              position: 'right',
              title: {
                display: true,
                text: 'Win Rate %'
              },
              max: 100,
              grid: {
                drawOnChartArea: false
              }
            }
          }
        }
      });

      setChartInstance(newChartInstance);
    }

    // Cleanup function
    return () => {
      if (chartInstance) {
        chartInstance.destroy();
      }
    };
  }, [strategies]);

  return (
    <div className="space-y-6">
      {/* System Status */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4">
        <h2 className="text-xl font-semibold mb-4">System Status</h2>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div className="bg-gray-50 dark:bg-gray-700 p-4 rounded-lg">
            <div className="text-sm text-gray-500 dark:text-gray-400">Connection Status</div>
            <div className="flex items-center mt-1">
              <div className={`w-3 h-3 rounded-full mr-2 ${connected ? 'bg-green-500' : 'bg-red-500'}`}></div>
              <div className="font-semibold">{connected ? 'Connected' : 'Disconnected'}</div>
            </div>
          </div>
          
          <div className="bg-gray-50 dark:bg-gray-700 p-4 rounded-lg">
            <div className="text-sm text-gray-500 dark:text-gray-400">Active Strategies</div>
            <div className="font-semibold mt-1">
              {strategies.filter(s => s.running).length} / {strategies.length}
            </div>
          </div>

          <div className="bg-gray-50 dark:bg-gray-700 p-4 rounded-lg">
            <div className="text-sm text-gray-500 dark:text-gray-400">Signals Generated</div>
            <div className="font-semibold mt-1">
              {strategies.reduce((total, s) => total + (s.signalsGenerated || 0), 0)}
            </div>
          </div>

          <div className="bg-gray-50 dark:bg-gray-700 p-4 rounded-lg">
            <div className="text-sm text-gray-500 dark:text-gray-400">Current Uptime</div>
            <div className="font-semibold mt-1">
              {strategies.length > 0 && strategies[0].startTime ? 
                formatUptime(new Date(strategies[0].startTime)) : 'N/A'}
            </div>
          </div>
        </div>
      </div>

      {/* Strategy Performance Chart */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4">
        <h2 className="text-xl font-semibold mb-4">Strategy Performance</h2>
        <div className="h-80">
          <canvas ref={chartRef}></canvas>
        </div>
      </div>

      {/* Strategy Details */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow">
        <div className="p-4 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-xl font-semibold">Strategy Details</h2>
        </div>
        
        {strategies.length === 0 ? (
          <div className="p-8 text-center text-gray-500 dark:text-gray-400">
            No active strategies found
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead className="bg-gray-50 dark:bg-gray-700">
                <tr>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                    Strategy
                  </th>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                    Status
                  </th>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                    Signals
                  </th>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                    P&L
                  </th>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                    Win Rate
                  </th>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                    Avg. Latency
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                {strategies.map((strategy, index) => (
                  <tr key={index} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                      {strategy.name || 'Unknown'}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      {strategy.running ? (
                        <span className="px-2 py-1 text-xs rounded-full bg-green-100 dark:bg-green-800 text-green-800 dark:text-green-100">
                          Active
                        </span>
                      ) : (
                        <span className="px-2 py-1 text-xs rounded-full bg-gray-100 dark:bg-gray-600 text-gray-800 dark:text-gray-200">
                          Inactive
                        </span>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      {strategy.signalsGenerated || 0}
                    </td>
                    <td className={`px-6 py-4 whitespace-nowrap text-sm font-medium ${
                      (strategy.profitLoss || 0) > 0 ? 'text-green-600 dark:text-green-400' : 
                      (strategy.profitLoss || 0) < 0 ? 'text-red-600 dark:text-red-400' : 
                      'text-gray-600 dark:text-gray-400'
                    }`}>
                      ${(strategy.profitLoss || 0).toFixed(2)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      {strategy.metrics && strategy.metrics.winRate 
                        ? `${(strategy.metrics.winRate * 100).toFixed(2)}%`
                        : 'N/A'}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      {strategy.metrics && strategy.metrics.averageLatency 
                        ? `${strategy.metrics.averageLatency.toFixed(2)} ms`
                        : 'N/A'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
};

// Helper function to format uptime
const formatUptime = (startTime) => {
  const now = new Date();
  const diffMs = now - startTime;
  
  const seconds = Math.floor(diffMs / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);
  
  if (days > 0) {
    return `${days}d ${hours % 24}h ${minutes % 60}m`;
  } else if (hours > 0) {
    return `${hours}h ${minutes % 60}m ${seconds % 60}s`;
  } else if (minutes > 0) {
    return `${minutes}m ${seconds % 60}s`;
  } else {
    return `${seconds}s`;
  }
};
