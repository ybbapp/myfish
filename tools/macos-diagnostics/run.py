"""Official macOS runner only. Never runs against a user's machine or box GUI."""
from pathlib import Path
import argparse,json,os,platform,subprocess,sys,time
p=argparse.ArgumentParser();p.add_argument('--ordinary',required=True);p.add_argument('--fixture',required=True);p.add_argument('--output',required=True);args=p.parse_args()
base=Path(args.output).resolve();base.mkdir(parents=True,exist_ok=True)
def dump(path,obj):path.write_text(json.dumps(obj,indent=2)+'\n')
def read(path):
    try:return json.loads(path.read_text())
    except (OSError,ValueError):return None
meta={'host':platform.platform(),'arch':platform.machine(),'sw_vers':subprocess.check_output(['sw_vers'],text=True),'display':subprocess.run(['system_profiler','SPDisplaysDataType'],capture_output=True,text=True).stdout,'fixture_is_instrumented':True,'native_screenshot_is_optional_not_assumed':True}
dump(base/'host.json',meta)
results=[]
def external_quit(pid):
    script=f'ObjC.import("AppKit"); var app=$.NSRunningApplication.runningApplicationWithProcessIdentifier({pid}); console.log("request_result="+app.terminate);'
    r=subprocess.run(['osascript','-l','JavaScript','-e',script],capture_output=True,text=True,timeout=10)
    return {'exit':r.returncode,'stdout':r.stdout,'stderr':r.stderr,'note':'exact-PID native termination request; boolean alone not treated as natural-exit proof'}
def run_case(name,binary,fixture=False,scale=.75):
    d=base/name;d.mkdir();env=os.environ.copy();env['TZ']='Asia/Shanghai'
    home=d/'home';home.mkdir();env['HOME']=str(home);env['MYFISH_DEBUG_INPUT']='0'
    data=home/'Library'/'Application Support'/'小鱼缸';data.mkdir(parents=True)
    settings={'mode':'light','opacity':.6,'highlight':.82,'scale':scale}
    dump(data/'settings.json',settings)
    env['MYFISH_DIAG_DIR']=str(d);env['MYFISH_DIAG_CASE']='/'.join(name.removeprefix('fixture-').rsplit('-',1))
    logfile=(d/'lifecycle.log').open('w');proc=subprocess.Popen([str(Path(binary).resolve())],env=env,stdout=logfile,stderr=subprocess.STDOUT)
    row={'name':name,'pid':proc.pid,'instrumented_fixture':fixture,'initial_scale':scale,'events_complete':False,'passed':False,'forced_cleanup':False};dump(d/'process.json',row)
    start=time.monotonic();seen=set();request=False
    try:
        while proc.poll() is None and time.monotonic()-start<30:
            if name.startswith('ordinary-') and not request and time.monotonic()-start>1.5:
                row['external_quit']=external_quit(proc.pid);request=True
            if name.startswith('fixture-external-') and not request and (d/'ready.json').exists():
                row['external_quit']=external_quit(proc.pid);request=True
            ready=read(d/'capture-ready.json')
            if ready and ready['step'] not in seen:
                step=ready['step'];info=read(d/(step+'.json'))
                if info:
                    # Native WindowServer capture. Never disable TCC or retry
                    # denied capture via another privileged route.
                    dest=d/(step+'-windowserver.png')
                    shot=subprocess.run(['screencapture','-x','-l',str(info['window_number']),str(dest)],capture_output=True,text=True,timeout=10)
                    dump(d/(step+'-windowserver-status.json'),{'exit':shot.returncode,'stderr':shot.stderr,'captured':shot.returncode==0 and dest.exists(),'native_window_id':info['window_number'],'backingScaleFactor':info['backingScaleFactor']})
                    (d/(step+'-capture-done')).write_text('native capture attempted; see status\n');seen.add(step)
            time.sleep(.08)
        if proc.poll() is None:
            row['timeout']=True
            subprocess.run(['sample',str(proc.pid),'1','-file',str(d/'hung-sample.txt')],capture_output=True,timeout=8)
            row['forced_cleanup']=True;proc.terminate()
            try:proc.wait(timeout=5)
            except subprocess.TimeoutExpired:proc.kill();proc.wait()
        else:proc.wait()
    except Exception as exc:
        row['runner_exception']=str(exc)
        if proc.poll() is None:
            row['forced_cleanup']=True;proc.terminate()
            try:proc.wait(timeout=5)
            except subprocess.TimeoutExpired:proc.kill();proc.wait()
    logfile.close()
    text=(d/'lifecycle.log').read_text(errors='replace')
    missing=[e for e in ['before-quit','will-quit','quit-complete','run-return'] if f'pid={proc.pid} event={e} ' not in text]
    row.update(exit=proc.returncode,missing_events=missing,events_complete=not missing,elapsed_seconds=round(time.monotonic()-start,2))
    row['exact_pid_gone']=subprocess.run(['ps','-p',str(proc.pid),'-o','pid='],capture_output=True,text=True).returncode!=0
    row['fixture_failure']=read(d/'failure.json')
    row['passed']=row['exit']==0 and not missing and row['exact_pid_gone'] and not row['forced_cleanup'] and not row['fixture_failure'] and not row.get('runner_exception')
    actual_settings=(d/'config'/'settings.json') if fixture else (data/'settings.json')
    row['settings_after']=read(actual_settings)
    dump(d/'verdict.json',row);results.append(row);dump(base/'summary.json',results)
    print(json.dumps(row),flush=True)
for scale in [.35,.75,1.25]:run_case('ordinary-'+str(scale),args.ordinary,scale=scale)
for route in ['main-api','delegate-close','appkit','menu','cmdq-equivalent','external']:
    # Split only the LAST '-' in fixture case -> route/state, allowing
    # descriptive hyphenated route names.
    for state in ['never','open','hidden']:
        run_case('fixture-'+route+'-'+state,args.fixture,fixture=True)
run_case('fixture-scale-never',args.fixture,fixture=True)
failed=[x['name'] for x in results if not x['passed']]
print('FAILED CASES:',failed,flush=True)
sys.exit(1 if failed else 0)
