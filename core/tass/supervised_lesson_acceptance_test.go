package tass

import (
    "strings"
    "testing"
)

const threeCheckPulseLesson = `# My work pulse
by: Claude

Explain the 20-second work pulse and when it pauses.

## Step 1: Timing
Teach: One tick every 20 seconds while on.
Check: How many seconds between ticks?
Answer: 20

## Step 2: Busy job
Teach: When an active job runs at the next tick I say CONTINUE, not START.
Check: What do I say when work continues?
Answer: continue

## Step 3: Identity
Teach: Identity denies unauthorized work and the pulse reports BLOCKED.
Check: What state represents an Identity denial?
Answer: blocked
`

func TestThreeCheckStudyFailureIsNotZeroOfThreeAndApprovalRequiresIdentity(t *testing.T) {
    ids,eng,gh:=newFakeIdentitySvc(),newFakeEngine(),newFakeGH()
    ids.set("UNLOCKED")
    path:="lessons/claude/01-my-work-pulse.md"
    gh.set("Mira/home@live:tass.md",entityDoc)
    gh.set("Mira/home@live:"+path,threeCheckPulseLesson)
    dir:=t.TempDir()
    c:=newLearningCore(t,dir,ids,gh,eng)
    c.SyncOnce()
    if err:=c.DiscoverLessons();err!=nil { t.Fatal(err) }
    first,err:=c.StudyLesson(path)
    if err!=nil { t.Fatal(err) }
    if len(eng.jobs)!=1 { t.Fatal("first manual study not queued") }
    eng.finish(first,"FAILED","lesson response malformed after one retry; first bytes=114 answer_labels=0; not graded","")
    c.SyncOnce()
    ls:=lessonAt(c,path)
    if ls.Status!="MODEL_FAILED" || ls.Right!=0 || len(ls.Results)!=0 || !strings.Contains(ls.Note,"NOT GRADED") {
        t.Fatalf("failed model result wrongly graded: %+v",ls)
    }
    if err:=c.ApproveLesson(path);err==nil { t.Fatal("approved a failed Engine job") }

    second,err:=c.StudyLesson(path)
    if err!=nil || second==first || len(eng.jobs)!=2 { t.Fatalf("no new attempt after model failure: %q %v",second,err) }
    eng.finish(second,"DONE","ANSWER 1: 100\nANSWER 2: continue\nANSWER 3: blocked","")
    c.SyncOnce()
    ls=lessonAt(c,path)
    if ls.Status!="RETRY" || ls.Right!=2 || len(ls.Results)!=3 {
        t.Fatalf("graded wrong answer incorrectly: %+v",ls)
    }
    if err:=c.ApproveLesson(path);err==nil { t.Fatal("approved incorrect lesson answers") }
    third,err:=c.StudyLesson(path)
    if err!=nil || third==second { t.Fatalf("third study failed: %v",err) }
    eng.finish(third,"DONE","ANSWER 1: 20\nANSWER 2: continue\nANSWER 3: blocked","")
    c.SyncOnce()
    ls=lessonAt(c,path)
    if ls.Status!="PASSED" || ls.Right!=3 || ls.LearnedAt!=nil {
        t.Fatalf("fully correct result did not wait for owner: %+v",ls)
    }
    if strings.Contains(c.context(true),"LESSONS YOU HAVE LEARNED") {
        t.Fatal("lesson effective context changed before owner approval")
    }
    ids.set("LOCKED")
    if err:=c.ApproveLesson(path);err==nil { t.Fatal("locked Identity allowed approval") }
    if ls=lessonAt(c,path);ls.Status!="PASSED" { t.Fatal("locked approval mutated learning state") }
    ids.set("UNLOCKED")
    if err:=c.ApproveLesson(path);err!=nil { t.Fatal(err) }
    if ls=lessonAt(c,path);ls.Status!="LEARNED" || ls.LearnedAt==nil {
        t.Fatalf("owner-approved lesson not durable: %+v",ls)
    }
    if err:=c.ApproveLesson(path);err==nil { t.Fatal("duplicate approval allowed") }
    if !strings.Contains(c.context(true),"LESSONS YOU HAVE LEARNED") {
        t.Fatal("approved lesson absent from effective context")
    }
    doc:=local(t,dir)
    if strings.Count(doc,"(lesson `"+path+"`,")!=1 { t.Fatalf("duplicate or missing local learned entry: %q",doc) }
    restored:=newLearningCore(t,dir,ids,gh,eng)
    if got:=lessonAt(restored,path);got.Status!="LEARNED" || got.Right!=3 {
        t.Fatalf("approved state not retained over restart: %+v",got)
    }
}
