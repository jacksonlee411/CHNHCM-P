#!/usr/bin/env node
// Render local Mermaid sources with an existing mermaid-cli installation.
// No package download, application execution, or external business operation.
import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { createRequire } from 'node:module';
import { createHash } from 'node:crypto';
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'..');
const args=process.argv.slice(2);
const flag=name=>args.includes(name);
const option=name=>{const i=args.indexOf(name);return i<0?undefined:args[i+1];};
if(flag('--help')){
  console.log('Usage: node render_diagrams.mjs --cli-root /path/to/@mermaid-js/mermaid-cli --browser /path/to/chrome [--only comma-separated-diagram-names] [--dry-run]');
  process.exit(0);
}
const directory=path.join(root,'diagrams');
const selected=option('--only');
const names=selected ? selected.split(',') : [];
const files=(await fs.readdir(directory)).filter(n=>n.endsWith('.mmd')&&(!selected||names.includes(n.slice(0,-4)))).sort();
if(selected && files.length!==new Set(names).size)throw new Error('Unknown selected diagram');
if(!files.length)throw new Error('No matching diagrams');
if(flag('--dry-run')){console.log(JSON.stringify({diagrams:files.length,writes:false}));process.exit(0);}
const cliRoot=option('--cli-root'),browserPath=option('--browser');
if(!cliRoot||!browserPath)throw new Error('--cli-root and --browser are required; use an existing installation');
const req=createRequire(path.join(path.resolve(cliRoot),'package.json'));
const puppeteer=(await import(pathToFileURL(req.resolve('puppeteer')).href)).default;
const {renderMermaid}=await import(pathToFileURL(path.join(path.resolve(cliRoot),'src/index.js')).href);
const version=JSON.parse(await fs.readFile(path.join(cliRoot,'package.json'),'utf8')).version;
const config=JSON.parse(await fs.readFile(path.join(directory,'mermaid-config.json'),'utf8'));
const browser=await puppeteer.launch({executablePath:browserPath,headless:true,args:['--no-sandbox','--disable-dev-shm-usage']});
const sha=value=>createHash('sha256').update(value).digest('hex');
const manifest={renderer:'@mermaid-js/mermaid-cli',version,browser:await browser.version(),generatedAt:new Date().toISOString(),partial:!!selected,diagrams:[]};
let failed=false;
try{
 for(const filename of files){
  const definition=await fs.readFile(path.join(directory,filename),'utf8');
  try{
   const {data,title}=await renderMermaid(browser,definition,'svg',{backgroundColor:'white',fontEmbed:false,mermaidConfig:config,svgId:'china-'+filename.slice(0,-4)});
   await fs.writeFile(path.join(directory,filename.replace(/\.mmd$/,'.svg')),data);
   manifest.diagrams.push({file:filename,title,sourceSha256:sha(definition),svgSha256:sha(data),bytes:data.length,status:'rendered'});
   console.log('rendered '+filename);
  }catch(error){
   failed=true;manifest.diagrams.push({file:filename,status:'failed',error:String(error)});
   console.error('FAILED '+filename+' '+String(error));
  }
 }
}finally{await browser.close();}
await fs.writeFile(path.join(root,'data',selected?'render-partial.json':'render-manifest.json'),JSON.stringify(manifest,null,2)+'\n');
if(failed)process.exitCode=1;
