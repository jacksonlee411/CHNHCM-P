import { useId, useState } from 'react';
import { IconButton, Menu, MenuItem, Tooltip } from '@mui/material';
import MoreVertIcon from '@mui/icons-material/MoreVert';

export type SecurityAction = {
  label: string;
  run: (trigger: HTMLButtonElement) => void;
};

export function SecurityActionsMenu({
  actions,
  triggerId,
}: {
  actions: SecurityAction[];
  triggerId?: string;
}) {
  const [anchor, setAnchor] = useState<HTMLButtonElement | null>(null);
  const generatedId = useId();
  const id = triggerId ?? generatedId;
  if (!actions.length) return null;
  return (
    <>
      <Tooltip title="更多操作">
        <IconButton
          size="small"
          aria-label="更多操作"
          aria-haspopup="menu"
          id={id}
          aria-controls={anchor ? id + '-menu' : undefined}
          aria-expanded={!!anchor}
          onClick={(e) => setAnchor(e.currentTarget)}
        >
          <MoreVertIcon />
        </IconButton>
      </Tooltip>
      <Menu
        id={id + '-menu'}
        anchorEl={anchor}
        open={!!anchor}
        onClose={() => setAnchor(null)}
        slotProps={{ list: { 'aria-labelledby': id } }}
      >
        {actions.map((action) => (
          <MenuItem
            key={action.label}
            onClick={() => {
              if (!anchor) return;
              setAnchor(null);
              action.run(anchor);
            }}
          >
            {action.label}
          </MenuItem>
        ))}
      </Menu>
    </>
  );
}
