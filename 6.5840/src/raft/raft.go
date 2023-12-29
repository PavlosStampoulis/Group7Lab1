package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	//	"bytes"

	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.5840/labgob"
	"6.5840/labrpc"
)

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}
type RaftState int

const (
	Follower RaftState = iota
	Candidate
	Leader
)

type LogEntry struct {
	Term    int
	Command interface{}
}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	currentTerm int
	votedFor    int
	log         []LogEntry

	commitIndex int //highest index known to be commited
	lastApplied int //Applied when known to be commited to a majority based on leaders commitIndex?

	applyCh chan ApplyMsg
	//Belongs to leader, reinitialize after election
	nextIndex  []int //next log entry leader will send to node at index (might be optimistic)
	matchIndex []int //index of highest log entry known to be replicated at node (only incremented when confirmed by node)

	//Additional, unsure if superfluous
	myState       RaftState
	leader        int
	timeSinceBeat int64
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	// Your code here (2A).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	term := rf.currentTerm
	isLeader := rf.myState == Leader
	return term, isLeader
}

/*
	save Raft's persistent state to stable storage,

where it can later be retrieved after a crash and restart.
see paper's Figure 2 for a description of what should be persistent.
before you've implemented snapshots, you should pass nil as the
second argument to persister.Save().
after you've implemented snapshots, pass the current snapshot
(or nil if there's not yet a snapshot).
*/
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// raftstate := w.Bytes()
	// rf.persister.Save(raftstate, nil)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).

}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int
	VoteGranted bool
}

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if args.Term < rf.currentTerm || args.LastLogIndex < rf.commitIndex {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	} else if (rf.votedFor == -1 || rf.votedFor == args.CandidateId || rf.currentTerm < args.Term) && rf.commitIndex <= args.LastLogIndex {
		rf.currentTerm = args.Term
		if rf.commitIndex < 0 {
			rf.timeSinceBeat = 0
			rf.votedFor = args.CandidateId
			rf.myState = Follower
			reply.Term = rf.currentTerm
			reply.VoteGranted = true
			return
		} else if rf.log[rf.commitIndex].Term <= args.LastLogTerm {
			rf.timeSinceBeat = 0
			rf.votedFor = args.CandidateId
			rf.myState = Follower
			reply.Term = rf.currentTerm
			reply.VoteGranted = true
			return
		}
		fmt.Println("Outside of all ifs")
	} else if rf.votedFor == rf.me && rf.currentTerm == args.Term && rf.commitIndex < args.LastLogIndex {
		rf.timeSinceBeat = 0 //I clearly shouldn't candidate first yo
	}
	rf.currentTerm = args.Term // already checked to be at least as big as our Term
	reply.Term = rf.currentTerm
	reply.VoteGranted = false
}

// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	isLeader := rf.myState == Leader
	index := len(rf.log)
	term := rf.currentTerm
	if isLeader {
		newEntry := LogEntry{term, command}
		rf.log = append(rf.log, newEntry)
		rf.matchIndex[rf.me] = index
	}

	// Your code here (2B).

	return index + 1, term, isLeader
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func (rf *Raft) ticker() {
	//fmt.Println(len(rf.peers))
	replies := make([]*RequestVoteReply, len(rf.peers))
	for rf.killed() == false {

		// Your code here (2A)
		if rf.myState == Candidate {
			yesVote := 0
			for _, rep := range replies {
				rf.mu.Lock()
				if rep.Term > rf.currentTerm {
					//this node is behind
					rf.myState = Follower
					rf.currentTerm = rep.Term
					rf.mu.Unlock()
					break
				} else if rep.Term == rf.currentTerm {
					//vote valid this term
					if rep.VoteGranted {
						yesVote++
					}
				}
				rf.mu.Unlock()
			}

			if yesVote > len(rf.peers)/2 {
				//congrats, you're leader
				//println(fmt.Sprint(rf.me) + " became leader?")
				rf.leadInit()
				rf.heartBeat()
			}
		}

		// Check if a leader election should be started.

		if rf.myState != Leader && atomic.LoadInt64(&rf.timeSinceBeat) > electionTime {

			rf.mu.Lock()
			rf.myState = Candidate
			rf.leader = -1
			rf.votedFor = rf.me
			rf.timeSinceBeat = 0
			rf.currentTerm += 1
			lastTerm := 0
			if rf.commitIndex >= 0 {
				lastTerm = rf.log[rf.commitIndex].Term
			}
			args := RequestVoteArgs{
				rf.currentTerm,
				rf.me,
				rf.commitIndex, //double check if lastApplied, log length or commitIndex
				lastTerm,
			}
			rf.mu.Unlock()
			for inx := range rf.peers {
				reply := RequestVoteReply{}
				replies[inx] = &reply
				if inx != rf.me {
					go rf.sendRequestVote(inx, &args, &reply)
				} else {
					rf.mu.Lock()
					replies[inx].Term = args.Term
					replies[inx].VoteGranted = true
					rf.mu.Unlock()
				}
			}
		}
		// pause for a random amount of time between 50 and 350
		// milliseconds.
		ms := 50 + (rand.Int63() % 400)
		rf.mu.Lock()
		rf.timeSinceBeat += ms
		rf.mu.Unlock()
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me
	rf.applyCh = applyCh
	// Your initialization code here (2A, 2B, 2C).
	rf.myState = Follower
	rf.votedFor = -1
	rf.commitIndex = -1
	rf.lastApplied = -1
	rf.nextIndex = make([]int, len(rf.peers))
	rf.matchIndex = make([]int, len(rf.peers))
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}

func (rf *Raft) leadInit() {
	rf.mu.Lock()
	rf.leader = rf.me
	rf.myState = Leader
	for idx := range rf.nextIndex {
		if 0 < rf.commitIndex {
			rf.nextIndex[idx] = rf.commitIndex //double check if lastApplied or commitIndex
		} else {
			rf.nextIndex[idx] = 0
		}

	}
	for idx := range rf.matchIndex {
		rf.matchIndex[idx] = -1
	}
	rf.matchIndex[rf.me] = rf.commitIndex
	rf.mu.Unlock()

}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

func (rf *Raft) heartBeat() {
	for rf.myState == Leader {

		//fmt.Println(fmt.Sprint(rf.me)+"s leader log: ", rf.log)
		//fmt.Println(fmt.Sprint(rf.me)+"s leader log len: ", len(rf.log))
		//fmt.Println(fmt.Sprint(rf.me)+"s leader commit: ", rf.commitIndex)
		for peer := range rf.peers {
			if peer == rf.me {
				continue
			}
			go rf.SendAppendEntries(peer)
		}
		//fmt.Println("Leader log: ", rf.log)
		rf.mu.Lock()
		i := rf.lastApplied + 1
		end := len(rf.log)
		for ; i < end; i++ {
			count := 0
			for _, mIndex := range rf.matchIndex {
				if i <= mIndex {
					count++
				}
			}
			if count < (len(rf.peers)+1)/2 {
				break
			}
			if rf.commitIndex < i {
				rf.commitIndex++

				report := ApplyMsg{}
				report.Command = rf.log[rf.commitIndex].Command
				report.CommandIndex = rf.commitIndex + 1
				report.CommandValid = true
				//fmt.Println("Leader sending report: ", report)
				rf.applyCh <- report
				rf.lastApplied = rf.commitIndex
			}

		}
		rf.mu.Unlock()
		time.Sleep(time.Duration(beatTime) * time.Millisecond)
	}
}

type AppendEntriesArgs struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}
type AppendEntriesReply struct {
	Term    int
	Success bool
}

// constants for sleep and timeouts
const beatTime = 110
const electionTime = 6 * beatTime

// handle AppendEntries RPC
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	//fmt.Println(fmt.Sprint(rf.me)+"s follower log: ", rf.log)
	//fmt.Println(fmt.Sprint(rf.me)+"s follower log len: ", len(rf.log))
	//fmt.Println(fmt.Sprint(rf.me)+"s follower commit: ", rf.commitIndex)
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		fmt.Println(fmt.Sprint(rf.me) + " wrong term")
		reply.Success = false

		return
	} else if args.LeaderCommit < rf.commitIndex {
		reply.Term = reply.Term - 1
		fmt.Println(fmt.Sprint(rf.me) + " leader is not up to date?!")
		reply.Success = false
		return
	}
	rf.currentTerm = args.Term
	rf.leader = args.LeaderId
	rf.myState = Follower
	rf.timeSinceBeat = 0
	if len(rf.log)-1 < args.PrevLogIndex {
		reply.Term = rf.currentTerm
		fmt.Println(fmt.Sprint(rf.me) + " log too short")
		reply.Success = false
		return
	} else if args.PrevLogIndex == -1 {
	} else if rf.log[args.PrevLogIndex].Term != args.PrevLogTerm {
		//fmt.Println("Before ", rf.log)
		if 1 < len(rf.log) {
			rf.log = rf.log[:args.PrevLogIndex]
		} else {
			rf.log = []LogEntry{}
		}
		//fmt.Println("After ", rf.log)
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}

	if len(args.Entries) == 0 {

	} else if 0 < len(rf.log) && args.PrevLogIndex != len(rf.log)-1 {
		if (args.PrevLogIndex+len(args.Entries)-1 < rf.commitIndex) && args.PrevLogIndex != -1 {
			fmt.Println("This is a fuckup?!")

			fmt.Println("has " + fmt.Sprint(rf.log[args.PrevLogIndex+1:]))
			fmt.Println("given " + fmt.Sprint(args.Entries))

		}
		rf.log = append(rf.log[:args.PrevLogIndex+1], args.Entries...)
	} else {
		rf.log = append(rf.log, args.Entries...)
	}

	reply.Term = rf.currentTerm
	reply.Success = true
	if args.LeaderCommit > rf.commitIndex {
		minInt := min(args.LeaderCommit, len(rf.log)-1)
		rf.commitIndex = minInt
	}
	if rf.commitIndex > rf.lastApplied {
		go rf.commitLog()
	}

}
func (rf *Raft) SendAppendEntries(peer int) {
	newLog := []LogEntry{} //make a log depending on peer's nextIndex
	lastTerm := 0
	rf.mu.Lock()
	lastIndex := rf.nextIndex[peer] - 1
	//fmt.Println("Leader's log: ", rf.log)
	currentIndex := len(rf.log) - 1
	sentTerm := rf.currentTerm
	if rf.nextIndex[peer] > 0 {
		lastTerm = rf.log[rf.nextIndex[peer]-1].Term
	}
	if rf.nextIndex[peer] <= len(rf.log)-1 {
		if 0 <= rf.nextIndex[peer] {
			newLog = rf.log[rf.nextIndex[peer]:]
		} else {
			newLog = rf.log
		}

	}
	args := AppendEntriesArgs{
		rf.currentTerm,
		rf.me,
		lastIndex,
		lastTerm,
		newLog,
		rf.commitIndex,
	}
	rf.mu.Unlock()
	reply := AppendEntriesReply{}
	ok := rf.peers[peer].Call("Raft.AppendEntries", &args, &reply)
	if !ok {
		return
		//fmt.Print("Handle Append Call Problem")
	}
	if !reply.Success {
		//handle append follower problem
		rf.mu.Lock()
		if rf.currentTerm < reply.Term || reply.Term < sentTerm {
			rf.leader = -1
			rf.votedFor = -1
			rf.myState = Follower
			rf.currentTerm = reply.Term
			rf.mu.Unlock()
			return
		} else if sentTerm != rf.currentTerm {
			rf.mu.Unlock()
			return
		} else {
			// followers log not up to date
			// double check if this will interfere with dealing with lock
			if rf.nextIndex[peer] <= 0 {

				rf.mu.Unlock()
				return
			}
			rf.nextIndex[peer]--
			rf.mu.Unlock()
			rf.SendAppendEntries(peer)
			return
		}
	} else {
		//ev. info applied, check if to update info about peer
		rf.mu.Lock()
		defer rf.mu.Unlock()
		rf.matchIndex[peer] = currentIndex
		rf.nextIndex[peer] = currentIndex + 1

	}

}

func (rf *Raft) commitLog() {
	rf.mu.Lock()
	i := rf.lastApplied + 1
	end := rf.commitIndex
	for ; i <= end; i++ {
		report := ApplyMsg{}
		report.Command = rf.log[i].Command
		report.CommandIndex = i + 1
		report.CommandValid = true
		//fmt.Println("Follower sending report: ", report)
		rf.applyCh <- report
		rf.lastApplied++
	}
	rf.mu.Unlock()

}
