import type { ReactNode } from 'react';
import { Box } from '@mui/material';
import './record-control.css';

export function RecordControl({ identity, actions, navigation, className = '' }: { identity: ReactNode; actions: ReactNode; navigation: ReactNode; className?: string }) {
  return <Box className={`record-control ${className}`}>
    <Box className="record-control-main">
      <Box className="record-control-identity">{identity}</Box>
      <Box className="record-control-actions">{actions}</Box>
      <Box className="record-control-navigation">{navigation}</Box>
    </Box>
  </Box>;
}
