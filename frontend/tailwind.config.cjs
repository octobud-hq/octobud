const plugin = require('tailwindcss/plugin');

module.exports = {
    content: ['./src/**/*.{html,js,svelte,ts}'],
    darkMode: 'class', // Enable class-based dark mode
    theme: {
        extend: {
            colors: {
                gray: {
                    // 950: '#101012' // Custom lighter shade for main background (default: #030712)
                }
            },
            boxShadow: {
                soft: '0 8px 24px rgba(15, 23, 42, 0.08)'
            }
        }
    },
    plugins: [
        // Plugins are now registered in CSS using @plugin directive
        // Keeping custom utility plugin here for scrollbar-hide
        plugin(function ({ addUtilities }) {
            addUtilities({
                '.scrollbar-hide': {
                    /* Firefox */
                    'scrollbar-width': 'none',
                    /* Safari and Chrome */
                    '&::-webkit-scrollbar': {
                        display: 'none'
                    }
                }
            })
        })
    ]
};

