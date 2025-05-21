package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Todo struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Title     string             `json:"title"`
	Completed bool               `json:"completed"`
}

var collection *mongo.Collection

func initMongo() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	clinet, err := mongo.NewClient(options.Client().ApplyURI(os.Getenv("MONGODB_URI")))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = clinet.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	collection = clinet.Database("mongo-todo-gin").Collection("todos")
}

func getTodos(c *gin.Context) {
	var todos []Todo
	cur, err := collection.Find(context.TODO(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	defer cur.Close(context.TODO())

	for cur.Next(context.TODO()) {
		var todo Todo

		cur.Decode(&todo)
		todos = append(todos, todo)
	}
	c.JSON(http.StatusOK, todos)
}

func getTodo(c *gin.Context) {
	id := c.Param("id")
	objId, _ := primitive.ObjectIDFromHex(id)
	var todo Todo
	err := collection.FindOne(context.TODO(), bson.M{"_id": objId}).Decode(&todo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, todo)
}

func createTodo(c *gin.Context) {
	var todo Todo
	if err := c.BindJSON(&todo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := collection.InsertOne(context.TODO(), todo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

func updateTodo(c *gin.Context) {
	id := c.Param("id")
	objID, _ := primitive.ObjectIDFromHex(id)

	var todo Todo
	if err := c.BindJSON(&todo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	update := bson.M{"$set": bson.M{"title": todo.Title, "completed": todo.Completed}}
	_, err := collection.UpdateOne(context.TODO(), bson.M{"_id": objID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Todo updated"})
}

func deleteTodo(c *gin.Context) {
	id := c.Param("id")
	objId, _ := primitive.ObjectIDFromHex(id)

	var todo Todo
	err := collection.FindOne(context.TODO(), bson.M{"_id": objId}).Decode(&todo)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
		return
	}

	_, err = collection.DeleteOne(context.TODO(), bson.M{"_id": objId})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Todo deleted"})
}

func main() {
	initMongo()

	r := gin.Default()

	r.GET("/todos", getTodos)
	r.GET("/todos/:id", getTodo)
	r.POST("/todos", createTodo)
	r.PUT("/todos/:id", updateTodo)
	r.DELETE("/todos/:id", deleteTodo)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
