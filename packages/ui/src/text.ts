const messages: Record<string, string> = {
  'Effective Dates': '生效日期', 'Select effective date': '选择生效日期',
  'Earlier effective dates': '较早的生效日期', 'Later effective dates': '较晚的生效日期',
  'Switch application': '切换应用', 'Applications': '应用', 'Application sections': '应用分组',
  'Open navigation': '打开导航', 'Close navigation': '关闭导航',
  'Choose an application and task.': '选择应用及办理事项。', 'To be designed': '待设计',
};
export const uiText = (key: string) => messages[key] ?? key;
