package foundation

// DefaultEffectiveDate is the initial choice for new configuration and master
// data (CHNHCM-202), not a minimum date, current day or historical founding date.
// Public write requests still require an explicit valid date.
const DefaultEffectiveDate = "1900-01-01"
