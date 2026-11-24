// Keep test identities and masked screenshots. Assertion values can contain login fields.
const { appendFileSync } = require('node:fs');
const path = require('node:path');

module.exports = class ReadyReporter {
  onBegin(config) {
    this.folder = path.dirname(config.projects[0].outputDir);
  }

  onTestEnd(test, result) {
    const screenshot = result.attachments.find(item => item.name === 'masked failure' && item.path);
    const record = {
      project: test.parent.project().name,
      id: test.id,
      title: test.title,
      status: result.status,
    };
    if (screenshot) record.screenshot = path.relative(this.folder, screenshot.path);
    appendFileSync(path.join(this.folder, 'browser.jsonl'), JSON.stringify(record) + '\n');
  }
};
