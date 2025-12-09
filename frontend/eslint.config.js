import eslintPluginSvelte from 'eslint-plugin-svelte';
import prettier from 'eslint-config-prettier';
import { fixupConfigRules } from '@eslint/compat';
import tsParser from '@typescript-eslint/parser';

export default [
	// Ignore patterns
	{
		ignores: [
			'**/.svelte-kit/**',
			'**/build/**',
			'**/node_modules/**',
			'**/.env/**',
			'**/dist/**',
			'**/static/sw.js' // Service worker - handled separately if needed
		]
	},
	// JavaScript files
	{
		files: ['**/*.{js,mjs,cjs}'],
		languageOptions: {
			ecmaVersion: 'latest',
			sourceType: 'module'
		},
		rules: {
			// Add any JS specific rules here
		}
	},
	// TypeScript files (including .d.ts)
	{
		files: ['**/*.ts'],
		languageOptions: {
			parser: tsParser,
			ecmaVersion: 'latest',
			sourceType: 'module',
			parserOptions: {
				project: './tsconfig.json'
			}
		},
		rules: {
			// Allow goto() calls in TS files - we use resolve() but with string concatenation
			'svelte/no-navigation-without-resolve': ['error', {
				ignoreGoto: true
			}]
		}
	},
	// Svelte files
	...fixupConfigRules(
		eslintPluginSvelte.configs['flat/recommended']
	),
	{
		files: ['**/*.svelte'],
		languageOptions: {
			parserOptions: {
				parser: tsParser
			}
		},
		rules: {
			// Svelte-specific rules
			'svelte/no-at-html-tags': 'warn',
			'svelte/valid-compile': 'error',
			// Allow href links to skip resolve() (external URLs don't need it)
			// Allow goto() calls - we use resolve() but with string concatenation for search params
			'svelte/no-navigation-without-resolve': ['error', {
				ignoreLinks: true,
				ignoreGoto: true
			}]
		}
	},
	// Prettier config (must be last to override other configs)
	prettier,
	prettier.configs?.recommended
].flat().filter(Boolean);
