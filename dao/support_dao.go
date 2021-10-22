package dao

import "go.mongodb.org/mongo-driver/bson"

// GetSupportChatUsers ...
func (dao FactoryDAO) GetSupportChatUsers() (interface{}, error) {
	var chats []bson.M

	group := bson.M{
		"$group": bson.M{
			"_id":      "$username",
			"user_id":  bson.M{"$first": "$user_id"},
			"username": bson.M{"$first": "$username"},
		},
	}
	project := bson.M{
		"$project": bson.M{
			"_id":      0,
			"username": 1,
			"user_id":  1,
		},
	}

	collection := dao.Collections["support_chat"]
	pipeline := []bson.M{group, project}
	cursor, err := collection.Aggregate(dao.ctx, pipeline)
	if err != nil {
		return nil, err
	}

	err = cursor.All(dao.ctx, &chats)

	return chats, err
}
