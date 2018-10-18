package cmd

import (
	"github.com/netlify/gotrue/conf"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage"
	"github.com/gobuffalo/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var autoconfirm, isSuperAdmin, isAdmin bool
var audience, instanceID string

func getAudience(c *conf.Configuration) string {
	if audience == "" {
		return c.JWT.Aud
	}

	return audience
}

func adminCmd() *cobra.Command {
	var adminCmd = &cobra.Command{
		Use: "admin",
	}

	adminCmd.AddCommand(&adminCreateUserCmd, &adminDeleteUserCmd)
	adminCmd.PersistentFlags().StringVarP(&audience, "aud", "a", "", "Set the new user's audience")
	adminCmd.PersistentFlags().StringVarP(&instanceID, "instance_id", "i", "", "Set the instance ID to interact with")

	adminCreateUserCmd.Flags().BoolVar(&autoconfirm, "confirm", false, "Automatically confirm user without sending an email")
	adminCreateUserCmd.Flags().BoolVar(&isSuperAdmin, "superadmin", false, "Create user with superadmin privileges")
	adminCreateUserCmd.Flags().BoolVar(&isAdmin, "admin", false, "Create user with admin privileges")

	return adminCmd
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

var adminDeleteUserCmd = cobra.Command{
	Use: "deleteuser",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			logrus.Fatal("Not enough arguments to deleteuser command. Expected at least ID or email")
			return
		}

		execWithConfigAndArgs(cmd, adminDeleteUser, args)
	},
}

var adminEditRoleCmd = cobra.Command{
	Use: "editrole",
	Run: func(cmd *cobra.Command, args []string) {
		execWithConfigAndArgs(cmd, adminEditRole, args)
	},
}

func adminCreateUser(globalConfig *conf.GlobalConfiguration, config *conf.Configuration, args []string) {
	iid := uuid.Must(uuid.FromString(instanceID))

	db, err := storage.Dial(globalConfig)
	if err != nil {
		logrus.Fatalf("Error opening database: %+v", err)
	}
	defer db.Close()

	aud := getAudience(config)
	if exists, err := models.IsDuplicatedEmail(db, iid, args[0], aud); exists {
		logrus.Fatalf("Error creating new user: user already exists")
	} else if err != nil {
		logrus.Fatalf("Error checking user email: %+v", err)
	}

	user, err := models.NewUser(iid, args[0], args[1], aud, nil)
	if err != nil {
		logrus.Fatalf("Error creating new user: %+v", err)
	}
	user.IsSuperAdmin = isSuperAdmin

	err = db.Transaction(func(tx *storage.Connection) error {
		var terr error
		if terr = tx.Create(user); terr != nil {
			return terr
		}

		if len(args) > 2 {
			if terr = user.SetRole(tx, args[2]); terr != nil {
				return terr
			}
		} else if isAdmin {
			if terr = user.SetRole(tx, config.JWT.AdminGroupName); terr != nil {
				return terr
			}
		}

		if config.Mailer.Autoconfirm || autoconfirm {
			if terr = user.Confirm(tx); terr != nil {
				return terr
			}
		}
		return nil
	})
	if err != nil {
		logrus.Fatalf("Unable to create user (%s): %+v", args[0], err)
	}

	logrus.Infof("Created user: %s", args[0])
}

func adminDeleteUser(globalConfig *conf.GlobalConfiguration, config *conf.Configuration, args []string) {
	iid := uuid.Must(uuid.FromString(instanceID))

	db, err := storage.Dial(globalConfig)
	if err != nil {
		logrus.Fatalf("Error opening database: %+v", err)
	}
	defer db.Close()

	user, err := models.FindUserByEmailAndAudience(db, iid, args[0], getAudience(config))
	if err != nil {
		userID := uuid.Must(uuid.FromString(args[0]))
		user, err = models.FindUserByInstanceIDAndID(db, iid, userID)
		if err != nil {
			logrus.Fatalf("Error finding user (%s): %+v", userID, err)
		}
	}

	if err = db.Destroy(user); err != nil {
		logrus.Fatalf("Error removing user (%s): %+v", args[0], err)
	}

	logrus.Infof("Removed user: %s", args[0])
}

func adminEditRole(globalConfig *conf.GlobalConfiguration, config *conf.Configuration, args []string) {
	iid := uuid.Must(uuid.FromString(instanceID))

	db, err := storage.Dial(globalConfig)
	if err != nil {
		logrus.Fatalf("Error opening database: %+v", err)
	}
	defer db.Close()

	user, err := models.FindUserByEmailAndAudience(db, iid, args[0], getAudience(config))
	if err != nil {
		userID := uuid.Must(uuid.FromString(args[0]))
		user, err = models.FindUserByInstanceIDAndID(db, iid, userID)
		if err != nil {
			logrus.Fatalf("Error finding user (%s): %+v", userID, err)
		}
	}

	user.IsSuperAdmin = isSuperAdmin

	if len(args) > 0 {
		user.Role = args[0]
	} else if isAdmin {
		user.Role = config.JWT.AdminGroupName
	}

	if err = db.UpdateOnly(user, "role", "is_super_admin"); err != nil {
		logrus.Fatalf("Error updating role for user (%s): %+v", args[0], err)
	}

	logrus.Infof("Updated user: %s", args[0])
}
