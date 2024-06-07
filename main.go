package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

var mg MongoInstance

const dbName = "fiber-hrms"
const mongoURI = "mongodb://localhost:27017/" + dbName
const employeeCollection = "employees"

type Employee struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty"`
	Name   string  `json:"name"`
	Salary float64 `json:"salary"`
	Age    float64 `json:"age"`
}

func connect() error {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	err = client.Connect(ctx)
	db := client.Database(dbName)

	if err != nil {
		return err
	}

	mg = MongoInstance{
		Client: client,
		Db:     db,
	}

	return nil
}

func main() {
	if err := connect(); err != nil {
		log.Fatal(err)
	}

	app := fiber.New()

	app.Get("/employee", func(c *fiber.Ctx) {
		query := bson.D{{}}

		cursor, err := mg.Db.Collection(employeeCollection).Find(c.Context(), query)
		if err != nil {
			c.Status(500).SendString(err.Error())
			return
		}

		var employees []Employee = make([]Employee, 0)
		if err := cursor.All(c.Context(), &employees); err != nil {
			c.Status(500).SendString(err.Error())
			return
		}

		c.JSON(employees)
	})

	app.Post("/employee", func(c *fiber.Ctx) {
		collection := mg.Db.Collection(employeeCollection)
		employee := new(Employee)

		if err := c.BodyParser(employee); err != nil {
			c.Status(400).SendString(err.Error())
			return
		}

		employee.ID = ""
		insertionResult, err := collection.InsertOne(c.Context(), employee)

		if err != nil {
			c.Status(500).SendString(err.Error())
			return
		}

		filter := bson.D{{Key: "_id", Value: insertionResult.InsertedID}}
		createdRecorded := collection.FindOne(c.Context(), filter)

		createdEmployee := &Employee{}
		createdRecorded.Decode(createdEmployee)
		c.Status(201).JSON(createdEmployee)
	})

	app.Put("/employee/:id", func(c *fiber.Ctx) {
		idParam := c.Params("id")
		employeeID, err := primitive.ObjectIDFromHex(idParam)

		if err != nil {
			c.Status(400)
			return
		}

		employee := new(Employee)
		if err := c.BodyParser(employee); err != nil {
			c.Status(400).SendString(err.Error())
			return
		}

		query := bson.D{{Key: "_id", Value: employeeID}}
		update := bson.D{
			{
				Key: "$set",
				Value: bson.D{
					{Key: "name", Value: employee.Name},
					{Key: "age", Value: employee.Age},
					{Key: "salary", Value: employee.Salary},
				},
			},
		}

		err = mg.Db.Collection(employeeCollection).FindOneAndUpdate(c.Context(), query, update).Err()

		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.SendStatus(400)
				return
			}
			c.SendStatus(500)
			return
		}

		employee.ID = idParam
		c.Status(201).JSON(employee)
	})

	app.Delete("/employee/:id", func(c *fiber.Ctx) {
		employeeID, err := primitive.ObjectIDFromHex(c.Params("id"))

		if err != nil {
			c.SendStatus(400)
			return
		}

		query := bson.D{{Key: "_id", Value: employeeID}}
		result, err := mg.Db.Collection(employeeCollection).DeleteOne(c.Context(), &query)

		if err != nil {
			c.SendStatus(500)
			return
		}

		if result.DeletedCount < 1 {
			c.SendStatus(404)
			return
		}

		c.Status(200).JSON("record deleted")
	})

	log.Fatal(app.Listen(":3000"))
}
