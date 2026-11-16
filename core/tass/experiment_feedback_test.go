package tass

import (
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
)

// These tests use Stage 3B's synthetic Engine and owner Identity; never the owner's data or Nemotron.
func feedbackReady(t *testing.T) (*Core, *fakeIdentitySvc, *runEngine, ExperimentView) {
	t.Helper()
	c, ids, re, plan := expSetup(t)
	if _, created, err := c.RunExperimentArm(plan.ExperimentID, "A", confirmOf(plan, "A")); err != nil || !created { t.Fatalf("A: %v", err) }
	waitArm(t, c, plan.ExperimentID, "A", "COMPLETED")
	if _, created, err := c.RunExperimentArm(plan.ExperimentID, "B", confirmOf(plan, "B")); err != nil || !created { t.Fatalf("B: %v", err) }
	return c, ids, re, waitArm(t, c, plan.ExperimentID, "B", "COMPLETED")
}

func TestOwnerFeedbackFourVerdictsAndReopen(t *testing.T) {
	for _, verdict := range []string{"BETTER","SAME","WORSE","INCONCLUSIVE"} {
		t.Run(verdict,func(t *testing.T){
			c, _, re, v := feedbackReady(t)
			thought := thoughtJSON(t,c)
			before := re.nonGet.Load()
			in := ExperimentFeedbackInput{Verdict:verdict,Correction:"The reply repeats evidence; try one specific new hypothesis."}
			f, created, err := c.SaveExperimentFeedback(v.ExperimentID,in)
			if err != nil || !created || f.Verdict != verdict || f.SnapshotSHA != v.Frozen.SHA256 {
				t.Fatalf("save: created %v err %v feedback %+v",created,err,f)
			}
			if after:=re.nonGet.Load(); after!=before {t.Fatalf("feedback made Engine writes: %d",after-before)}
			if after:=thoughtJSON(t,c);after!=thought {t.Fatal("feedback changed the Thought")}
			again, created, err := c.SaveExperimentFeedback(v.ExperimentID,in)
			if err!=nil || created || again.SavedAt!=f.SavedAt {t.Fatalf("idempotency: %v %v %+v",created,err,again)}
			if _,_,err:=c.SaveExperimentFeedback(v.ExperimentID,ExperimentFeedbackInput{Verdict:"WORSE",Correction:"different"});!errors.Is(err,ErrExperimentConflict) && verdict!="WORSE" {
				t.Fatalf("conflicting save: %v",err)
			}
			reopen,err:=c.ExperimentByID(v.ExperimentID)
			if err!=nil || reopen.Feedback==nil || reopen.Feedback.Verdict!=verdict || reopen.Feedback.EvidenceSHA!=f.EvidenceSHA {
				t.Fatalf("reopen: %v %+v",err,reopen.Feedback)
			}
			fi,err:=os.Stat(c.expRunPath(v.ExperimentID,experimentFeedbackFile))
			if err!=nil || fi.Mode().Perm()!=0o600 {t.Fatalf("feedback file private: %v %v",fi,err)}
		})
	}
}

func TestOwnerFeedbackRejectsIncompleteUnverifiedAndInvalid(t *testing.T) {
	c,ids,re,p:=expSetup(t)
	for _, in := range []ExperimentFeedbackInput{
		{Verdict:"BETTER"}, {Verdict:"SAME"}, {Verdict:"WORSE"}, {Verdict:"INCONCLUSIVE"},
	} {
		if _,_,err:=c.SaveExperimentFeedback(p.ExperimentID,in);!errors.Is(err,ErrExperimentFeedbackIncomplete) {t.Fatalf("unrun %q: %v",in.Verdict,err)}
	}
	c.RunExperimentArm(p.ExperimentID,"A",confirmOf(p,"A"))
	waitArm(t,c,p.ExperimentID,"A","COMPLETED")
	if _,_,err:=c.SaveExperimentFeedback(p.ExperimentID,ExperimentFeedbackInput{Verdict:"INCONCLUSIVE"});!errors.Is(err,ErrExperimentFeedbackIncomplete) {t.Fatalf("only A: %v",err)}
	c.RunExperimentArm(p.ExperimentID,"B",confirmOf(p,"B"))
	waitArm(t,c,p.ExperimentID,"B","COMPLETED")
	for _, in := range []ExperimentFeedbackInput{
		{Verdict:"GOOD"}, {Verdict:"SAME",Correction:strings.Repeat("x",501)},
		{Verdict:"SAME",Correction:"bad\x07control"},
	} {
		if _,_,err:=c.SaveExperimentFeedback(p.ExperimentID,in);!errors.Is(err,ErrExperimentInvalid) {t.Fatalf("invalid %+v: %v",in,err)}
	}
	re.mu.Lock();re.fx.tamper=true;re.mu.Unlock()
	if _,_,err:=c.SaveExperimentFeedback(p.ExperimentID,ExperimentFeedbackInput{Verdict:"SAME"});!errors.Is(err,ErrExperimentFeedbackIncomplete) {t.Fatalf("tampered Engine snapshot: %v",err)}
	re.mu.Lock();re.fx.tamper=false;re.mu.Unlock()
	ids.set("LOCKED")
	if _,_,err:=c.SaveExperimentFeedback(p.ExperimentID,ExperimentFeedbackInput{Verdict:"SAME"});err==nil {t.Fatal("locked Identity accepted")}
	if _,err:=os.Stat(c.expRunPath(p.ExperimentID,experimentFeedbackFile));!os.IsNotExist(err) {t.Fatalf("refused save wrote file: %v",err)}
}

func TestOwnerFeedbackFailsClosedOnCorruptOrUnsafeFile(t *testing.T) {
	for _, tc:=range []struct{name string; edit func(t *testing.T,path string)}{
		{"corrupt",func(t *testing.T,path string){os.WriteFile(path,[]byte("{broken}\n"),0o600)}},
		{"oversized",func(t *testing.T,path string){os.WriteFile(path,[]byte(strings.Repeat("x",experimentFeedbackMax+1)),0o600)}},
		{"public mode",func(t *testing.T,path string){os.Chmod(path,0o644)}},
		{"symlink",func(t *testing.T,path string){symlinkToCopy(t,path)}},
		{"wrong owner",func(t *testing.T,path string){rewriteJSON(t,path,func(m map[string]any){m["owner_entity_id"]="ent_other"})}},
	} {
		t.Run(tc.name,func(t *testing.T){
			c,_,_,v:=feedbackReady(t)
			in:=ExperimentFeedbackInput{Verdict:"SAME",Correction:"checked"}
			if _,_,err:=c.SaveExperimentFeedback(v.ExperimentID,in);err!=nil {t.Fatal(err)}
			path:=c.expRunPath(v.ExperimentID,experimentFeedbackFile)
			tc.edit(t,path)
			if _,err:=c.ExperimentByID(v.ExperimentID);!errors.Is(err,ErrExperimentFeedbackUnsafe) {t.Fatalf("unsafe GET: %v",err)}
			if _,_,err:=c.SaveExperimentFeedback(v.ExperimentID,in);!errors.Is(err,ErrExperimentFeedbackUnsafe) {t.Fatalf("unsafe POST: %v",err)}
		})
	}
}

func TestOwnerFeedbackRejectsOtherOwnerAndUnknownExperiment(t *testing.T) {
	c, ids, re, v := feedbackReady(t)
	before := re.nonGet.Load()
	ids.mu.Lock()
	ids.entityID = "ent_other"
	ids.mu.Unlock()
	if _,_,err:=c.SaveExperimentFeedback(v.ExperimentID,ExperimentFeedbackInput{Verdict:"SAME"});err==nil {
		t.Fatal("another owner saved an evaluation using a direct experiment ID")
	}
	if _,err:=c.ExperimentByID(v.ExperimentID);err==nil {
		t.Fatal("another owner reopened private feedback")
	}
	ids.mu.Lock()
	ids.entityID = "ent_0123"
	ids.mu.Unlock()
	if _,_,err:=c.SaveExperimentFeedback("exp_00000000000000000000",ExperimentFeedbackInput{Verdict:"SAME"});!errors.Is(err,ErrExperimentNotFound) {
		t.Fatalf("unknown experiment: %v",err)
	}
	if _,err:=os.Stat(c.expRunPath(v.ExperimentID,experimentFeedbackFile));!os.IsNotExist(err) {
		t.Fatalf("refused access wrote feedback: %v",err)
	}
	if re.nonGet.Load()!=before { t.Fatal("a refused feedback request wrote to Engine") }
}

func TestOwnerFeedbackConcurrentSingleWriter(t *testing.T) {
	c,_,_,v:=feedbackReady(t)
	type outcome struct{ created bool; err error }
	ch:=make(chan outcome,2)
	var wg sync.WaitGroup
	for _, verdict:=range []string{"SAME","WORSE"} {
		wg.Add(1)
		go func(s string){defer wg.Done();_,created,err:=c.SaveExperimentFeedback(v.ExperimentID,ExperimentFeedbackInput{Verdict:s});ch<-outcome{created,err}}(verdict)
	}
	wg.Wait();close(ch)
	created,conflicts:=0,0
	for r:=range ch {if r.created && r.err==nil {created++} else if errors.Is(r.err,ErrExperimentConflict) {conflicts++} else {t.Errorf("unexpected concurrent result: %+v",r)}}
	if created!=1 || conflicts!=1 {t.Fatalf("created %d, conflicts %d",created,conflicts)}
}
