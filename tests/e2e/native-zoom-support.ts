import { chromium, expect } from '@playwright/test';
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { localConnection } from '../../scripts/local-runtime.mjs';

// Set the browser's native tab zoom. A private extension avoids dependence on
// X11 keystroke injection; CSS zoom and device scale are never substituted.
export async function nativeZoomBrowser() {
  const directory=mkdtempSync(join(tmpdir(),'chnhcm-native-zoom-'));
  const extension=join(directory,'extension');mkdirSync(extension);
  writeFileSync(join(extension,'manifest.json'),JSON.stringify({manifest_version:3,name:'CHNHCM native zoom verification',version:'1.0',permissions:['tabs'],background:{service_worker:'worker.js'}}));
  writeFileSync(join(extension,'worker.js'),'chrome.runtime.onInstalled.addListener(()=>{});');
  let context:Awaited<ReturnType<typeof chromium.launchPersistentContext>>|undefined;
  try {
    context=await chromium.launchPersistentContext(join(directory,'profile'),{channel:'chromium',headless:true,baseURL:localConnection().webOrigin,viewport:{width:1280,height:900},args:[`--disable-extensions-except=${extension}`,`--load-extension=${extension}`]});
    const worker=context.serviceWorkers()[0]??await context.waitForEvent('serviceworker');
    const page=await context.newPage();
    return {page,context,
      async setZoom(factor:number) {
        await worker.evaluate(async({factor,url})=>{const chrome=(globalThis as any).chrome;const tabs=await chrome.tabs.query({});const tab=tabs.find((t:any)=>t.url===url);if(!tab)throw Error('missing isolated zoom tab');await chrome.tabs.setZoom(tab.id,factor)}, {factor,url:page.url()});
        await expect.poll(()=>page.evaluate(()=>({dpr:devicePixelRatio,width:innerWidth,height:innerHeight}))).toEqual({dpr:factor,width:1280/factor,height:900/factor});
      },
      async close(){try{await context?.close()}finally{rmSync(directory,{recursive:true,force:true})}}
    };
  } catch(error){await context?.close();rmSync(directory,{recursive:true,force:true});throw error;}
}
