package live

import (
	"api/question"
	"math/rand"
)

type QwizOptions struct {
	ShuffleQuestions bool
	ShuffleAnswers   bool
}

type QwizParticipant struct {
	ID                int32
	DisplayName       string
	ProfilePictureURI string
}

type StartingLiveQwiz struct{}

type RunningLiveQwiz struct {
	QuestionNumber  int
	CurrentAnswers  []string
	CorrectAnswer   uint8
	AcceptedAnswers map[int32][]bool
}

func NewRunningLiveQwiz(question question.Question, shuffleAnswers bool, participantIDs []int32) *RunningLiveQwiz {
	var answers []string
	answers = append(answers, question.Answer1, question.Answer2)
	if question.Answer3 != nil {
		answers = append(answers, *question.Answer3)
	}
	if question.Answer4 != nil {
		answers = append(answers, *question.Answer4)
	}

	indices := make([]uint8, len(answers))
	for i := range indices {
		indices[i] = uint8(i)
	}

	if shuffleAnswers {
		rand.Shuffle(len(indices), func(i, j int) {
			indices[i], indices[j] = indices[j], indices[i]
		})
	}

	var correctAnswer uint8
	for idx, val := range indices {
		if int16(val) == question.Correct-1 {
			correctAnswer = uint8(idx)
			break
		}
	}

	currentAnswers := make([]string, len(indices))
	for i, idx := range indices {
		currentAnswers[i] = answers[idx]
	}

	acceptedAnswers := make(map[int32][]bool)
	for _, id := range participantIDs {
		acceptedAnswers[id] = []bool{}
	}

	return &RunningLiveQwiz{
		QuestionNumber:  0,
		CurrentAnswers:  currentAnswers,
		CorrectAnswer:   correctAnswer,
		AcceptedAnswers: acceptedAnswers,
	}
}

func (s *RunningLiveQwiz) Next(nextQuestion *question.Question, shuffleAnswers bool, participantIDs []int32) QwizState {
	if nextQuestion != nil {
		nq := *nextQuestion
		questionNumber := s.QuestionNumber + 1
		nextAnswers := []string{nq.Answer1, nq.Answer2}
		if nq.Answer3 != nil {
			nextAnswers = append(nextAnswers, *nq.Answer3)
		}
		if nq.Answer4 != nil {
			nextAnswers = append(nextAnswers, *nq.Answer4)
		}

		indices := make([]uint8, len(nextAnswers))
		for i := range indices {
			indices[i] = uint8(i)
		}

		if shuffleAnswers {
			rand.Shuffle(len(indices), func(i, j int) {
				indices[i], indices[j] = indices[j], indices[i]
			})
		}

		var correctAnswer uint8
		for idx, val := range indices {
			if int16(val) == nq.Correct-1 {
				correctAnswer = uint8(idx)
				break
			}
		}

		currentAnswers := make([]string, len(indices))
		for i, idx := range indices {
			currentAnswers[i] = nextAnswers[idx]
		}

		acceptedAnswers := make(map[int32][]bool)
		for id, answers := range s.AcceptedAnswers {
			newAnswers := append([]bool(nil), answers...)
			for len(newAnswers) < questionNumber {
				newAnswers = append(newAnswers, false)
			}
			acceptedAnswers[id] = newAnswers
		}

		return &RunningLiveQwiz{
			QuestionNumber:  questionNumber,
			CurrentAnswers:  currentAnswers,
			CorrectAnswer:   correctAnswer,
			AcceptedAnswers: acceptedAnswers,
		}
	}

	finishing := &FinishingLiveQwiz{}
	return finishing.fromRunning(s)
}

type FinishingLiveQwiz struct {
	Participants []int32
	Scores       map[int32]uint
}

func (s *FinishingLiveQwiz) fromRunning(r *RunningLiveQwiz) QwizState {
	scores := make(map[int32]int)

	for id, answers := range r.AcceptedAnswers {
		score := 0
		for _, answer := range answers {
			if answer {
				score++
			}
		}
		scores[id] = score
	}

	// Convert scores to map[int32]uint
	uintScores := make(map[int32]uint)
	for id, score := range scores {
		uintScores[id] = uint(score)
	}

	s.Scores = uintScores // Assuming FinishingLiveQwiz has a Scores field.
	return s
}

const (
	StartingState = iota
	RunningState
	FinishingState
)

type Qwiz struct {
	QwizID       int32
	Options      QwizOptions
	Questions    chan question.Question
	Participants []QwizParticipant
	State        QwizState
}

func newLiveQwiz(qwizID int32, options QwizOptions, questions []question.Question, participants []QwizParticipant) *Qwiz {
	questionChan := make(chan question.Question, len(questions))
	for _, q := range questions {
		questionChan <- q
	}
	close(questionChan) // Закрываем канал после отправки всех вопросов

	return &Qwiz{
		QwizID:       qwizID,
		Options:      options,
		Questions:    questionChan,
		Participants: participants,
		State:        &StartingLiveQwiz{},
	}
}

type QwizState interface {
	IsLiveQwizState() bool
	Next(question *question.Question, shuffleAnswers bool, participantIDs []int32) QwizState
}

func (s *StartingLiveQwiz) Next(question *question.Question, shuffleAnswers bool, participantIDs []int32) QwizState {
	if question != nil {
		// Если у нас есть следующий вопрос, переходим в состояние RunningLiveQwiz
		return NewRunningLiveQwiz(*question, shuffleAnswers, participantIDs)
	}

	// Если вопросов больше нет, переходим в состояние FinishingLiveQwiz
	return NewFinishingLiveQwiz(participantIDs)
}

func (s *FinishingLiveQwiz) Next(question *question.Question, shuffleAnswers bool, participantIDs []int32) QwizState {
	return s
}

func (s *StartingLiveQwiz) IsLiveQwizState() bool {
	return true
}

func (s *RunningLiveQwiz) IsLiveQwizState() bool {
	return true
}

func (s *FinishingLiveQwiz) IsLiveQwizState() bool {
	return true
}

func NewFinishingLiveQwiz(participantIDs []int32) *FinishingLiveQwiz {
	return &FinishingLiveQwiz{
		Participants: participantIDs,
	}
}

func (l *Qwiz) Progress(question *question.Question) {
	l.State = l.State.Next(question, l.Options.ShuffleAnswers, getParticipantIDs(l.Participants))
}

func getParticipantIDs(participants []QwizParticipant) []int32 {
	ids := make([]int32, len(participants))
	for i, p := range participants {
		ids[i] = p.ID
	}
	return ids
}
