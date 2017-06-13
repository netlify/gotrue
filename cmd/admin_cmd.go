package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage/dial"
	"github.com/spf13/cobra"
)

var autoconfirm, isSuperAdmin, isAdmin bool
var audience string

func getAudience(c *conf.Configuration) string {
	if audience == "" {
		return c.JWT.Aud
	}

	return audience
}

func init() {
	adminCmd.AddCommand(&adminCreateUserCmd)
	adminCmd.PersistentFlags().StringVarP(&audience, "aud", "a", "", "Set the new user's audience")

	adminCreateUserCmd.Flags().BoolVar(&autoconfirm, "confirm", false, "Automatically confirm user without sending an email")
	adminCreateUserCmd.Flags().BoolVar(&isSuperAdmin, "superadmin", false, "Create user with superadmin privileges")
	adminCreateUserCmd.Flags().BoolVar(&isAdmin, "admin", false, "Create user with admin privileges")
}

var adminCmd = cobra.Command{
	Use: "admin",
}

var adminCreateUserCmd = cobra.Command{
	Use: "createuser",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			logrus.Fatal("Not enough arguments to createuser command. Expected at least email and password values")
			return
		}

		execWithConfigAndArgs(cmd, adminCreateUser, args)
	},
}

func adminCreateUser(config *conf.Configuration, args []string) {
	db, err := dial.Dial(config)
	if err != nil {
		logrus.Fatalf("Error opening database: %+v", err)
	}

	if config.DB.Automigrate {
		if err := db.Automigrate(); err != nil {
			logrus.Fatalf("Error migrating tables: %+v", err)
		}
	}

	user, err := models.NewUser(args[0], args[1], getAudience(config), nil)
	if err != nil {
		logrus.Fatalf("Error creating new user: %+v", err)
	}

	if len(args) > 2 {
		user.SetRole(args[2])
	} else if isAdmin {
		user.SetRole("admin")
	}

	user.IsSuperAdmin = isSuperAdmin

	if err := db.CreateUser(user); err != nil {
		logrus.Fatalf("Unable to create user (%s): %+v", args[0], err)
		return
	}

	logrus.Infof("Created user: %s", args[0])
}
