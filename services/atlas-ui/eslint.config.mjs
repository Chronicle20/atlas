import nextConfig from "eslint-config-next";

const eslintConfig = [
  ...nextConfig,
  {
    rules: {
      // Allow setState in effects for common data fetching patterns
      'react-hooks/set-state-in-effect': 'off',
      // Allow Math.random in useMemo for skeleton loading states
      'react-hooks/purity': 'off',
    },
  },
];

export default eslintConfig;
