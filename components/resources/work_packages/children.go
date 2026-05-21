package work_packages

import (
	"fmt"

	"github.com/opf/openproject-cli/components/parser"
	"github.com/opf/openproject-cli/components/paths"
	"github.com/opf/openproject-cli/components/requests"
	"github.com/opf/openproject-cli/dtos"
	"github.com/opf/openproject-cli/models"
)

// Children returns all direct children of the given work package.
func Children(parentId uint64) ([]*models.WorkPackage, error) {
	query := requests.NewUnpaginatedQuery(nil, []requests.Filter{
		{Operator: "=", Name: "parent", Values: []string{fmt.Sprintf("%d", parentId)}},
	})

	response, err := requests.Get(paths.WorkPackages(), &query)
	if err != nil {
		return nil, err
	}

	collection := parser.Parse[dtos.WorkPackageCollectionDto](response)
	return collection.Convert().Items, nil
}
