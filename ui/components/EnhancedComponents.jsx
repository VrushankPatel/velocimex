// Enhanced UI Components for Velocimex
// Note: This is a JSX file for reference, but we'll implement these as vanilla JS

// Market Card Component
export const MarketCard = ({ symbol, exchange, price, change, isActive, onClick }) => {
    return `
        <div class="market-item ${isActive ? 'active' : ''}" data-symbol="${symbol}" onclick="${onClick}">
            <div>
                <div class="market-symbol">${symbol}</div>
                <div class="text-xs text-slate-500 dark:text-slate-400">${exchange}</div>
            </div>
            <div class="text-right">
                <div class="market-price">$${price.toFixed(2)}</div>
                <div class="market-change ${change >= 0 ? 'positive' : 'negative'}">
                    ${change >= 0 ? '+' : ''}${change.toFixed(2)}%
                </div>
            </div>
        </div>
    `;
};

// Order Book Row Component
export const OrderBookRow = ({ price, size, total, isBest, side }) => {
    const colorClass = side === 'ask' ? 'text-red-600 dark:text-red-400' : 'text-green-600 dark:text-green-400';
    const highlightClass = isBest ? (side === 'ask' ? 'best-ask' : 'best-bid') : '';
    
    return `
        <div class="orderbook-row ${highlightClass}">
            <div class="${colorClass}">${price.toFixed(2)}</div>
            <div>${size.toFixed(4)}</div>
            <div>${total.toFixed(4)}</div>
        </div>
    `;
};

// Arbitrage Opportunity Component
export const ArbitrageCard = ({ symbol, buyExchange, sellExchange, profitPercent, maxVolume }) => {
    return `
        <div class="arbitrage-item">
            <div class="arbitrage-symbol">${symbol}</div>
            <div class="arbitrage-exchanges">${buyExchange} → ${sellExchange}</div>
            <div class="flex justify-between items-center">
                <div class="arbitrage-profit ${profitPercent >= 0 ? 'positive' : 'negative'}">
                    ${profitPercent.toFixed(2)}%
                </div>
                <div class="text-xs text-slate-500 dark:text-slate-400">
                    Vol: ${maxVolume.toFixed(2)}
                </div>
            </div>
        </div>
    `;
};

// Trading Signal Component
export const SignalCard = ({ symbol, side, quantity, price, timestamp }) => {
    const timeAgo = getTimeAgo(new Date(timestamp));
    
    return `
        <div class="signal-item">
            <div class="signal-icon ${side.toLowerCase()}">
                ${side === 'BUY' ? 'B' : 'S'}
            </div>
            <div class="signal-content">
                <div class="signal-symbol">${symbol}</div>
                <div class="signal-details">
                    ${side} ${quantity} @ $${price}
                </div>
            </div>
            <div class="signal-time">
                ${timeAgo}
            </div>
        </div>
    `;
};

// Notification Component
export const NotificationItem = ({ title, message, type, timestamp, isRead }) => {
    const timeAgo = getTimeAgo(timestamp);
    const icon = getNotificationIcon(type);
    
    return `
        <div class="notification-item ${isRead ? 'opacity-60' : ''}">
            <i data-lucide="${icon}" class="notification-icon"></i>
            <div class="notification-content">
                <div class="notification-title">${title}</div>
                <div class="notification-message">${message}</div>
                <div class="notification-time">${timeAgo}</div>
            </div>
        </div>
    `;
};

// Toast Component
export const Toast = ({ message, type, duration = 5000 }) => {
    const icon = getToastIcon(type);
    
    return `
        <div class="toast ${type} fade-in">
            <div class="flex items-center">
                <i data-lucide="${icon}" class="w-5 h-5 mr-2"></i>
                <span>${message}</span>
            </div>
        </div>
    `;
};

// Performance Metric Component
export const MetricCard = ({ label, value, trend, color = 'default' }) => {
    const trendIcon = trend > 0 ? 'trending-up' : trend < 0 ? 'trending-down' : 'minus';
    const trendColor = trend > 0 ? 'text-green-600' : trend < 0 ? 'text-red-600' : 'text-slate-500';
    
    return `
        <div class="flex justify-between items-center">
            <span class="text-sm text-slate-600 dark:text-slate-400">${label}</span>
            <div class="flex items-center space-x-1">
                <span class="font-semibold text-slate-900 dark:text-slate-100">${value}</span>
                ${trend !== 0 ? `
                    <i data-lucide="${trendIcon}" class="w-3 h-3 ${trendColor}"></i>
                ` : ''}
            </div>
        </div>
    `;
};

// Loading Skeleton Component
export const LoadingSkeleton = ({ type = 'default', count = 3 }) => {
    const skeletons = {
        market: () => `
            <div class="animate-pulse space-y-3">
                ${Array(count).fill().map(() => `
                    <div class="h-16 bg-slate-200 dark:bg-slate-700 rounded-lg"></div>
                `).join('')}
            </div>
        `,
        orderbook: () => `
            <div class="animate-pulse space-y-1">
                ${Array(count).fill().map(() => `
                    <div class="h-6 bg-slate-200 dark:bg-slate-700 rounded"></div>
                `).join('')}
            </div>
        `,
        arbitrage: () => `
            <div class="animate-pulse space-y-3">
                ${Array(count).fill().map(() => `
                    <div class="h-20 bg-slate-200 dark:bg-slate-700 rounded-lg"></div>
                `).join('')}
            </div>
        `,
        signal: () => `
            <div class="animate-pulse space-y-3">
                ${Array(count).fill().map(() => `
                    <div class="h-16 bg-slate-200 dark:bg-slate-700 rounded-lg"></div>
                `).join('')}
            </div>
        `,
        default: () => `
            <div class="animate-pulse space-y-2">
                ${Array(count).fill().map(() => `
                    <div class="h-4 bg-slate-200 dark:bg-slate-700 rounded w-3/4"></div>
                `).join('')}
            </div>
        `
    };
    
    return skeletons[type] ? skeletons[type]() : skeletons.default();
};

// Utility functions
function getTimeAgo(timestamp) {
    const now = new Date();
    const diff = now - timestamp;
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);

    if (minutes < 1) return 'Just now';
    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    return `${days}d ago`;
}

function getNotificationIcon(type) {
    const icons = {
        success: 'check-circle',
        error: 'x-circle',
        warning: 'alert-triangle',
        info: 'info',
        trade: 'trending-up',
        arbitrage: 'zap'
    };
    return icons[type] || 'info';
}

function getToastIcon(type) {
    const icons = {
        success: 'check-circle',
        error: 'x-circle',
        warning: 'alert-triangle',
        info: 'info'
    };
    return icons[type] || 'info';
}

// Chart configuration presets
export const ChartPresets = {
    performance: {
        type: 'line',
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: { display: false },
                tooltip: {
                    mode: 'index',
                    intersect: false,
                    backgroundColor: 'rgba(0, 0, 0, 0.8)',
                    titleColor: 'white',
                    bodyColor: 'white',
                    borderColor: 'rgba(255, 255, 255, 0.1)',
                    borderWidth: 1
                }
            },
            scales: {
                x: {
                    type: 'time',
                    time: {
                        unit: 'minute',
                        displayFormats: { minute: 'HH:mm' }
                    },
                    grid: { color: 'rgba(148, 163, 184, 0.1)' },
                    ticks: { color: 'rgb(148, 163, 184)' }
                },
                y: {
                    grid: { color: 'rgba(148, 163, 184, 0.1)' },
                    ticks: {
                        color: 'rgb(148, 163, 184)',
                        callback: function(value) {
                            return '$' + value.toFixed(2);
                        }
                    }
                }
            },
            interaction: {
                mode: 'nearest',
                axis: 'x',
                intersect: false
            },
            elements: {
                point: { radius: 0, hoverRadius: 6 }
            }
        }
    },
    
    volume: {
        type: 'bar',
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: { display: false }
            },
            scales: {
                x: {
                    grid: { color: 'rgba(148, 163, 184, 0.1)' },
                    ticks: { color: 'rgb(148, 163, 184)' }
                },
                y: {
                    grid: { color: 'rgba(148, 163, 184, 0.1)' },
                    ticks: { color: 'rgb(148, 163, 184)' }
                }
            }
        }
    }
};

// Theme utilities
export const ThemeUtils = {
    applyTheme(theme) {
        if (theme === 'dark') {
            document.documentElement.classList.add('dark');
        } else {
            document.documentElement.classList.remove('dark');
        }
        localStorage.setItem('theme', theme);
    },
    
    getCurrentTheme() {
        return document.documentElement.classList.contains('dark') ? 'dark' : 'light';
    },
    
    toggleTheme() {
        const current = this.getCurrentTheme();
        const newTheme = current === 'dark' ? 'light' : 'dark';
        this.applyTheme(newTheme);
        return newTheme;
    }
};

// Animation utilities
export const AnimationUtils = {
    fadeIn(element, duration = 300) {
        element.style.opacity = '0';
        element.style.transition = `opacity ${duration}ms ease-in`;
        element.offsetHeight; // Trigger reflow
        element.style.opacity = '1';
    },
    
    fadeOut(element, duration = 300) {
        element.style.transition = `opacity ${duration}ms ease-out`;
        element.style.opacity = '0';
        setTimeout(() => element.remove(), duration);
    },
    
    slideIn(element, direction = 'up', duration = 300) {
        const transforms = {
            up: 'translateY(20px)',
            down: 'translateY(-20px)',
            left: 'translateX(20px)',
            right: 'translateX(-20px)'
        };
        
        element.style.opacity = '0';
        element.style.transform = transforms[direction];
        element.style.transition = `all ${duration}ms ease-out`;
        element.offsetHeight; // Trigger reflow
        element.style.opacity = '1';
        element.style.transform = 'translate(0, 0)';
    }
};
