package tasks

import (
	"github.com/merico-dev/lake/models/domainlayer/ticket"
	"github.com/merico-dev/lake/plugins/core"
	"github.com/merico-dev/lake/plugins/helper"
	"github.com/merico-dev/lake/plugins/tapd/models"
	"reflect"
)

func ConvertIssueAssigneeHistory(taskCtx core.SubTaskContext) error {
	data := taskCtx.GetData().(*TapdTaskData)
	logger := taskCtx.GetLogger()
	db := taskCtx.GetDb()
	logger.Info("convert board:%d", data.Options.WorkspaceId)

	cursor, err := db.Model(&models.TapdIssueAssigneeHistory{}).Where("source_id = ? AND workspace_id = ?", data.Source.ID, data.Options.WorkspaceId).Rows()
	if err != nil {
		return err
	}
	defer cursor.Close()
	converter, err := helper.NewDataConverter(helper.DataConverterArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: TapdApiParams{
				SourceId: data.Source.ID,
				//CompanyId:   data.Source.CompanyId,
				WorkspaceId: data.Options.WorkspaceId,
			},
			Table: "tapd_api_%",
		},
		InputRowType: reflect.TypeOf(models.TapdIssueAssigneeHistory{}),
		Input:        cursor,
		Convert: func(inputRow interface{}) ([]interface{}, error) {
			toolL := inputRow.(*models.TapdIssueAssigneeHistory)
			domainL := &ticket.IssueAssigneeHistory{
				IssueId:   IssueIdGen.Generate(data.Source.ID, toolL.IssueId),
				Assignee:  toolL.Assignee,
				StartDate: toolL.StartDate,
				EndDate:   &toolL.EndDate,
			}
			return []interface{}{
				domainL,
			}, nil
		},
	})
	if err != nil {
		return err
	}

	return converter.Execute()
}

var ConvertIssueAssigneeHistoryMeta = core.SubTaskMeta{
	Name:             "convertIssueAssigneeHistory",
	EntryPoint:       ConvertIssueAssigneeHistory,
	EnabledByDefault: true,
	Description:      "convert Tapd IssueAssigneeHistory",
}
