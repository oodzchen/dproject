package store

import (
	"errors"
	"log"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/oodzchen/dproject/config"
	mt "github.com/oodzchen/dproject/mocktool"
	"github.com/oodzchen/dproject/store/pgstore"
	"golang.org/x/crypto/bcrypt"
)

func connectDB() (*pgstore.PGStore, error) {
	err := config.Init("testdata/.env.local")
	if err != nil {
		return nil, err
	}

	appCfg := config.Config

	// fmt.Printf("App config: %#v\n", appCfg)
	// if appCfg.Debug {
	// 	utils.PrintJSONf("App config:\n", appCfg)
	// }
	// fmt.Println("DSN: ", os.Getenv("DB_DSN"))

	pg := pgstore.New(&pgstore.DBConfig{
		DSN: appCfg.DB.GetDSN(),
	})

	err = pg.ConnectDB()
	if err != nil {
		return nil, err
	}

	return pg, nil
}

func registerNewUser(store *Store) (int, error) {
	user := mt.GenUser()
	pwd, _ := bcrypt.GenerateFromPassword([]byte(config.Config.DB.UserDefaultPassword), 10)
	return store.User.Create(user.Email, string(pwd), user.Name)
}

func createNewArticle(store *Store, userId int) (int, error) {
	article := mt.GenArticle()
	return store.Article.Create(article.Title, article.Content, userId, 0)
}

func TestArticleVote(t *testing.T) {
	pg, err := connectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer pg.CloseDB()

	store, err := New(pg)
	if err != nil {
		log.Fatal(err)
	}

	uId, err := registerNewUser(store)
	mt.LogFailed(err)

	aId, err := createNewArticle(store, uId)
	mt.LogFailed(err)

	uBId, err := registerNewUser(store)
	mt.LogFailed(err)

	// fmt.Println("uId: ", uId)
	// fmt.Println("aId: ", aId)
	// fmt.Println("uBId: ", uBId)

	t.Run("Vote up", func(t *testing.T) {
		err = store.Article.Vote(aId, uBId, "up")
		if err != nil {
			t.Errorf("should vote up success but got %v", err)
		}
	})

	t.Run("Change vote to down", func(t *testing.T) {
		err = store.Article.Vote(aId, uBId, "down")
		if err != nil {
			t.Errorf("should vote down success but got %v", err)
		}
	})

	t.Run("Revoke vote", func(t *testing.T) {
		err = store.Article.Vote(aId, uBId, "down")
		if err != nil {
			t.Errorf("should revoke vote success but got %v", err)
		}
	})
}

func TestArticleCheckVote(t *testing.T) {
	pg, err := connectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer pg.CloseDB()

	store, err := New(pg)
	if err != nil {
		log.Fatal(err)
	}

	uId, err := registerNewUser(store)
	mt.LogFailed(err)

	aId, err := createNewArticle(store, uId)
	mt.LogFailed(err)

	t.Run("Check unvote article", func(t *testing.T) {
		err, _ = store.Article.VoteCheck(aId, uId)
		// fmt.Println("vote check result: ", err)
		if !errors.Is(err, pgx.ErrNoRows) {
			t.Errorf("vote check should get %v, but get %v", pgx.ErrNoRows, err)
		}
	})

	t.Run("Check voted article", func(t *testing.T) {
		err = store.Article.Vote(aId, uId, "down")
		if err != nil {
			t.Errorf("vote down article failed: %v", err)
		}

		err, vt := store.Article.VoteCheck(aId, uId)
		if err != nil {
			t.Errorf("should check with no error, but got: %v", err)
		}

		if vt != "down" {
			t.Errorf("should get vote type as 'down' but got %s", vt)
		}
	})
}
