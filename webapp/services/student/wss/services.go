package wss

import (
	"fmt"
	"log"
	"time"

	"github.com/philippecery/libs/crng"
	"github.com/philippecery/maths/webapp/constant/homework"
	"github.com/philippecery/maths/webapp/constant/operation"
	"github.com/philippecery/maths/webapp/database/dataaccess"
	"github.com/philippecery/maths/webapp/database/model"
)

func (s *socket) operation() error {
	if session := s.getHomeworkSession(); session != nil {
		if homeworkType, exists := homework.Types[session.Homework.Type]; exists {
			operatorIDs := session.OperatorIDs()
			if len(operatorIDs) > 0 {
				fmt.Printf("Remaining operators: %v\n", operatorIDs)
				rndIdx, _ := crng.GetNumber(len(operatorIDs))
				s.operationCount = s.operationCount + 1
				nextOperation := &model.Operation{OperationID: s.operationCount, OperatorID: operatorIDs[rndIdx], Status: operation.Todo}
				var operandRange *homework.OperandRanges
				switch nextOperation.OperatorID {
				case 1:
					operandRange = homeworkType.AdditionRange
				case 2:
					operandRange = homeworkType.SubstractionRange
				case 3:
					operandRange = homeworkType.MultiplicationRange
				case 4:
					operandRange = homeworkType.DivisionRange
				}
				nextOperation.Operand1, _ = crng.GetNumberInRange(operandRange.Operand1.RangeMin, operandRange.Operand1.RangeMax)
				nextOperation.Operand2, _ = crng.GetNumberInRange(operandRange.Operand2.RangeMin, operandRange.Operand2.RangeMax)
				s.addOperation(nextOperation)
				operator := homework.Operators[nextOperation.OperatorID]
				return s.emitTextMessage(map[string]interface{}{
					"response":      "operation",
					"operationName": s.getLocalizedMessage(operator.I18N),
					"operand1":      nextOperation.Operand1,
					"operand2":      nextOperation.Operand2,
					"operator":      operator.Symbol,
				})
			}
			log.Printf("/student/websocket[operation]: session completed")
		} else {
			log.Printf("/student/websocket[operation]: Invalid type ID")
		}
	} else {
		log.Printf("/student/websocket[operation]: No HomeworkSession found in database")
	}
	return s.emitErrorMessage("errorGenericMessage")
}

func (s *socket) answer() error {
	if session := s.getHomeworkSession(); session != nil {
		o := session.GetCurrentOperation()
		var answer, answer2 int
		answer, _ = s.toInt("answer")
		if o.OperatorID == 4 {
			answer2, _ = s.toInt("answer2")
		}
		good := o.VerifyResult(answer, answer2)
		nbUpdate := session.NbUpdate(good, o.OperatorID)
		percentUpdate := (nbUpdate * 100) / session.Homework.NumberOfOperationsByOperator(o.OperatorID)
		var percentAll int
		if good {
			o.Status = operation.Good
			percentAll = (session.NbTotalGood() * 100) / session.Homework.NumberOfOperations()
		} else {
			o.Status = operation.Wrong
		}
		nbTotalRemaining := session.Homework.NumberOfOperations() - session.NbTotalGood()
		s.saveAnswer(o)
		return s.emitTextMessage(map[string]interface{}{
			"response":         "answer",
			"good":             good,
			"nbUpdate":         nbUpdate,
			"percentUpdate":    percentUpdate,
			"percentAll":       percentAll,
			"nbTotalRemaining": nbTotalRemaining,
		})
	}
	log.Printf("/student/websocket[answer]: No HomeworkSession found in database")
	return s.emitErrorMessage("errorGenericMessage")
}

func (s *socket) toggle() error {
	if session := s.getHomeworkSession(); session != nil {
		if show, err := s.getBool("show"); err == nil {
			operation := session.GetCurrentOperation()
			if show {
				result, result2 := operation.GetResult()
				return s.emitTextMessage(map[string]interface{}{
					"response":   "toggle",
					"showResult": true,
					"result":     result,
					"result2":    result2,
				})
			}
			answer, answer2 := operation.GetAnswer()
			return s.emitTextMessage(map[string]interface{}{
				"response":   "toggle",
				"showResult": false,
				"answer":     answer,
				"answer2":    answer2,
			})
		}
		log.Printf("/student/websocket[toggle]: Invalid 'show' parameter")
	} else {
		log.Printf("/student/websocket[toggle]: No HomeworkSession found in database")
	}
	return s.emitErrorMessage("errorGenericMessage")
}

func (s *socket) end() error {
	if session := s.getHomeworkSession(); session != nil {
		session.EndTime = time.Now()
		session.Status = homework.Cancel
		var timeout bool
		var err error
		if timeout, err = s.getBool("timeout"); err == nil {
			if timeout {
				session.Status = homework.Timeout
			} else {
				session.Status = homework.Success
				for o := len(session.Operations) - 1; o >= 0; o-- {
					if session.Operations[o].Status == operation.Wrong {
						session.Status = homework.Failed
						break
					}
				}
			}
		}
		dataaccess.EndHomeworkSession(session.SessionID, session.EndTime, session.Status)
		s.summary(session)
		return nil
	}
	log.Printf("/student/websocket[end]: No HomeworkSession found in database")
	return s.emitErrorMessage("errorGenericMessage")
}

func (s *socket) results(homeworkType, status, page int) error {
	response := map[string]interface{}{"response": "results", "nbTotal": 0}
	sessions := make([]interface{}, 0)
	if homeworkSessions, nbTotal := dataaccess.GetSessionsByUserID(s.userID, homeworkType, status, page); homeworkSessions != nil {
		response["nbTotal"] = nbTotal
		for _, homeworkSession := range homeworkSessions {
			session := map[string]interface{}{
				"sessionID":         homeworkSession.SessionID,
				"startTime":         homeworkSession.FormattedDateTime(s.getCurrentLanguage()),
				"type":              homeworkSession.Homework.Type.Logo(),
				"nbAdditions":       homeworkSession.Homework.NbAdditions,
				"nbSubstractions":   homeworkSession.Homework.NbSubstractions,
				"nbMultiplications": homeworkSession.Homework.NbMultiplications,
				"nbDivisions":       homeworkSession.Homework.NbDivisions,
				"duration":          homeworkSession.FormattedDuration(),
				"status":            homeworkSession.Status.Logo(),
			}
			sessions = append(sessions, session)
		}
	}
	response["sessions"] = sessions
	return s.emitTextMessage(response)
}

func (s *socket) details(sessionID string) error {
	if session := dataaccess.GetSessionByID(sessionID); session != nil {
		s.summary(session)
		return nil
	}
	log.Printf("/student/websocket[end]: HomeworkSession %s not found in DB", sessionID)
	return s.emitErrorMessage("errorGenericMessage")
}

func (s *socket) summary(session *model.HomeworkSession) {
	if nbTotal := session.Homework.NbAdditions; nbTotal > 0 {
		if err := s.emitTextMessage(map[string]interface{}{
			"response":      "summary",
			"operationName": s.getLocalizedMessage("additions"),
			"nbTotal":       nbTotal,
			"nbGood":        session.Additions.NbGood,
			"nbWrong":       session.Additions.NbWrong,
			"operationsTodo": s.getLocalizedMessage("summaryOperationsTodo", nbTotal, map[string]interface{}{
				"nbTotal":        nbTotal,
				"operationsType": s.getLocalizedMessage("Addition", nbTotal),
			}),
			"operationsGood": s.getLocalizedMessage("summaryOperationsGood", session.Additions.NbGood, map[string]interface{}{
				"nbGood": session.Additions.NbGood,
			}),
			"operationsWrong": s.getLocalizedMessage("summaryOperationsWrong", session.Additions.NbWrong, map[string]interface{}{
				"nbWrong": session.Additions.NbWrong,
			}),
		}); err != nil {
			log.Printf("Unable to emit response \"summary\" for operation \"additions\". Cause: %s\n", err)
		}
	}
	if nbTotal := session.Homework.NbSubstractions; nbTotal > 0 {
		if err := s.emitTextMessage(map[string]interface{}{
			"response":      "summary",
			"operationName": s.getLocalizedMessage("substractions"),
			"nbTotal":       nbTotal,
			"nbGood":        session.Substractions.NbGood,
			"nbWrong":       session.Substractions.NbWrong,
			"operationsTodo": s.getLocalizedMessage("summaryOperationsTodo", nbTotal, map[string]interface{}{
				"nbTotal":        nbTotal,
				"operationsType": s.getLocalizedMessage("Substraction", nbTotal),
			}),
			"operationsGood": s.getLocalizedMessage("summaryOperationsGood", session.Substractions.NbGood, map[string]interface{}{
				"nbGood": session.Substractions.NbGood,
			}),
			"operationsWrong": s.getLocalizedMessage("summaryOperationsWrong", session.Substractions.NbWrong, map[string]interface{}{
				"nbWrong": session.Substractions.NbWrong,
			}),
		}); err != nil {
			log.Printf("Unable to emit response \"summary\" for operation \"substractions\". Cause: %s\n", err)
		}
	}
	if nbTotal := session.Homework.NbMultiplications; nbTotal > 0 {
		if err := s.emitTextMessage(map[string]interface{}{
			"response":      "summary",
			"operationName": s.getLocalizedMessage("multiplications"),
			"nbTotal":       nbTotal,
			"nbGood":        session.Multiplications.NbGood,
			"nbWrong":       session.Multiplications.NbWrong,
			"operationsTodo": s.getLocalizedMessage("summaryOperationsTodo", nbTotal, map[string]interface{}{
				"nbTotal":        nbTotal,
				"operationsType": s.getLocalizedMessage("Multiplication", nbTotal),
			}),
			"operationsGood": s.getLocalizedMessage("summaryOperationsGood", session.Multiplications.NbGood, map[string]interface{}{
				"nbGood": session.Multiplications.NbGood,
			}),
			"operationsWrong": s.getLocalizedMessage("summaryOperationsWrong", session.Multiplications.NbWrong, map[string]interface{}{
				"nbWrong": session.Multiplications.NbWrong,
			}),
		}); err != nil {
			log.Printf("Unable to emit response \"summary\" for operation \"multiplications\". Cause: %s\n", err)
		}
	}
	if nbTotal := session.Homework.NbDivisions; nbTotal > 0 {
		if err := s.emitTextMessage(map[string]interface{}{
			"response":      "summary",
			"operationName": s.getLocalizedMessage("divisions"),
			"nbTotal":       nbTotal,
			"nbGood":        session.Divisions.NbGood,
			"nbWrong":       session.Divisions.NbWrong,
			"operationsTodo": s.getLocalizedMessage("summaryOperationsTodo", nbTotal, map[string]interface{}{
				"nbTotal":        nbTotal,
				"operationsType": s.getLocalizedMessage("Division", nbTotal),
			}),
			"operationsGood": s.getLocalizedMessage("summaryOperationsGood", session.Divisions.NbGood, map[string]interface{}{
				"nbGood": session.Divisions.NbGood,
			}),
			"operationsWrong": s.getLocalizedMessage("summaryOperationsWrong", session.Divisions.NbWrong, map[string]interface{}{
				"nbWrong": session.Divisions.NbWrong,
			}),
		}); err != nil {
			log.Printf("Unable to emit response \"summary\" for operation \"divisions\". Cause: %s\n", err)
		}
	}
}
