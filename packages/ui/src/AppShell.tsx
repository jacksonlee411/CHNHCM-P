import type { ReactNode } from 'react';
import { AppBar, Box, CssBaseline, ThemeProvider, Toolbar, Typography } from '@mui/material';
import { theme } from './theme';
import './layout.css';

export function AppShell({ navigation, account, notices, children }: {
  navigation?: ReactNode; account?: ReactNode; notices?: ReactNode; children: ReactNode;
}) {
  return <ThemeProvider theme={theme}>
    <CssBaseline />
    <a className="skip-content" href="#main-content">跳到正文</a>
    <AppBar position="fixed" className="app-header">
      <Toolbar sx={{ minHeight: 'var(--chnhcm-header-height) !important', gap: 1, minWidth: 0 }}>
        {navigation ?? <Typography variant="h6">CHNHCM</Typography>}
        {account && <Box className="app-account">{account}</Box>}
      </Toolbar>
    </AppBar>
    <Box component="main" id="main-content" tabIndex={-1} className="app-content">
      {notices && <Box className="app-notices">{notices}</Box>}
      {children}
    </Box>
  </ThemeProvider>;
}
