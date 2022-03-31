package tasks

import (
	"fmt"
	"github.com/merico-dev/lake/models/domainlayer"
	"github.com/merico-dev/lake/models/domainlayer/didgen"
	"github.com/merico-dev/lake/models/domainlayer/ticket"
	"github.com/merico-dev/lake/plugins/core"
	"github.com/merico-dev/lake/plugins/helper"
	"github.com/merico-dev/lake/plugins/tapd/models"
	"reflect"
	"strconv"
)

func ConvertBug(taskCtx core.SubTaskContext) error {
	data := taskCtx.GetData().(*TapdTaskData)
	logger := taskCtx.GetLogger()
	db := taskCtx.GetDb()
	logger.Info("convert board:%d", data.Options.WorkspaceId)
	issueIdGen := didgen.NewDomainIdGenerator(&models.TapdBug{})
	cursor, err := db.Model(&models.TapdBug{}).Where("source_id = ? AND workspace_id = ?", data.Source.ID, data.Options.WorkspaceId).Rows()
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
			Table: RAW_BUG_TABLE,
		},
		InputRowType: reflect.TypeOf(models.TapdBug{}),
		Input:        cursor,
		Convert: func(inputRow interface{}) ([]interface{}, error) {
			toolL := inputRow.(*models.TapdBug)
			domainL := &ticket.Issue{
				DomainEntity: domainlayer.DomainEntity{
					Id: issueIdGen.Generate(toolL.SourceId, toolL.ID),
				},
				Url:            fmt.Sprintf("https://www.tapd.cn/%d/prong/Bugs/view/%d", toolL.WorkspaceID, toolL.ID),
				Key:            strconv.FormatUint(toolL.ID, 10),
				Title:          toolL.Title,
				Summary:        toolL.Title,
				EpicKey:        toolL.EpicKey,
				Type:           "BUG",
				Status:         toolL.Status,
				ResolutionDate: toolL.Resolved,
				CreatedDate:    toolL.Created,
				UpdatedDate:    toolL.Modified,
				ParentIssueId:  issueIdGen.Generate(toolL.SourceId, toolL.IssueID),
				Priority:       toolL.Priority,
				CreatorId:      UserIdGen.Generate(data.Options.SourceId, toolL.WorkspaceID, toolL.Reporter),
				AssigneeId:     UserIdGen.Generate(data.Options.SourceId, toolL.WorkspaceID, toolL.De),
				AssigneeName:   toolL.De,
				Severity:       toolL.Severity,
				Component:      toolL.Feature, // todo not sure about this
			}
			if domainL.ResolutionDate != nil && domainL.CreatedDate != nil {
				domainL.TimeSpentMinutes = int64(domainL.ResolutionDate.Minute() - domainL.CreatedDate.Minute())
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

var ConvertBugMeta = core.SubTaskMeta{
	Name:             "convertBug",
	EntryPoint:       ConvertBug,
	EnabledByDefault: true,
	Description:      "convert Tapd Bug",
}
