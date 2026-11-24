import type { ReactNode } from 'react'
import { Box, Stack, Typography } from '@mui/material'

interface PageHeaderProps {
  actions?: ReactNode
  headingComponent?: 'h1' | 'h2' | 'h3' | 'h4'
  leading?: ReactNode
  subtitle?: string
  surface?: 'page' | 'preview'
  testId?: string
  title: string
  titleId?: string
  titleTabIndex?: number
}

export function PageHeader({
  actions,
  headingComponent,
  leading,
  subtitle,
  surface = 'page',
  testId,
  title,
  titleId,
  titleTabIndex
}: PageHeaderProps) {
  if (surface === 'preview') {
    return (
      <Stack
        alignItems={{ md: 'center', xs: 'stretch' }}
        data-testid={testId}
        direction={{ md: 'row', xs: 'column' }}
        justifyContent='space-between'
        spacing={1}
        sx={{ mb: 1, minWidth: 0 }}
      >
        <Stack
          alignItems='center'
          direction='row'
          spacing={1}
          sx={{ flex: { md: '1 1 0', xs: '0 1 auto' }, minWidth: 0 }}
        >
          {leading}
          <Box sx={{ maxWidth: '100%', minWidth: 0 }}>
            <Typography
              component={headingComponent ?? 'h3'}
              id={titleId}
              sx={{ overflowWrap: 'anywhere' }}
              tabIndex={titleTabIndex}
              variant='h6'
            >
              {title}
            </Typography>
            {subtitle ? (
              <Typography color='text.secondary' sx={{ overflowWrap: 'anywhere' }} variant='body2'>
                {subtitle}
              </Typography>
            ) : null}
          </Box>
        </Stack>
        {actions ? (
          <Stack
            direction='row'
            flexWrap='wrap'
            justifyContent={{ md: 'flex-end', xs: 'flex-start' }}
            spacing={0.5}
            sx={{ flex: { md: '0 0 auto', xs: '0 1 auto' }, maxWidth: '100%', minWidth: 0 }}
          >
            {actions}
          </Stack>
        ) : null}
      </Stack>
    )
  }

  return (
    <Stack
      alignItems={{ md: 'center', xs: 'flex-start' }}
      data-testid={testId}
      direction={{ md: 'row', xs: 'column' }}
      justifyContent='space-between'
      spacing={2}
      sx={{ mb: 2, minWidth: 0 }}
    >
      <Stack
        alignItems='center'
        direction='row'
        spacing={1}
        sx={{ flex: { md: '1 1 0', xs: '0 1 auto' }, maxWidth: '100%', minWidth: 0 }}
      >
        {leading}
        <Box sx={{ maxWidth: '100%', minWidth: 0 }}>
          <Typography component={headingComponent ?? 'h2'} id={titleId} sx={{ overflowWrap: 'anywhere' }} tabIndex={titleTabIndex} variant='h5'>
            {title}
          </Typography>
          {subtitle ? (
            <Typography color='text.secondary' sx={{ overflowWrap: 'anywhere' }} variant='body2'>
              {subtitle}
            </Typography>
          ) : null}
        </Box>
      </Stack>
      {actions ? (
        <Stack
          direction='row'
          spacing={1}
          sx={{ flex: { md: '0 0 auto', xs: '0 1 auto' }, maxWidth: '100%', minWidth: 0 }}
        >
          {actions}
        </Stack>
      ) : null}
    </Stack>
  )
}
