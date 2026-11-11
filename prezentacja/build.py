#!/usr/bin/env python3
# Buduje prezentacja.html: osadza fonty Latin Modern jako data URI.
import base64, re, os, sys
here=os.path.dirname(os.path.abspath(__file__))
FONTDIR='/usr/share/texmf/fonts/opentype/public/lm/'
src=open(os.path.join(here,'prezentacja.src.html'),encoding='utf-8').read()
def rep(m):
    f=m.group(1); data=base64.b64encode(open(FONTDIR+f,'rb').read()).decode()
    return f'url("data:font/otf;base64,{data}") format("opentype")'
out=re.sub(r'url\("FONT:([^"]+)"\)',rep,src)
dst=os.path.join(here,'prezentacja.html')
open(dst,'w',encoding='utf-8').write(out)
print('ok',dst,len(out))
