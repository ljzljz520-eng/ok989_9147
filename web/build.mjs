import {cp,mkdir} from 'node:fs/promises';await mkdir('dist',{recursive:true});await cp('index.html','dist/index.html');await cp('app.js','dist/app.js');
