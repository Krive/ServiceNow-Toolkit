package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow/identity"
)

var identityCmd = &cobra.Command{
	Use:   "identity",
	Short: "Identity and access management operations",
	Long:  "Manage ServiceNow users, roles, groups, and access control",
}

// User commands
var userCmd = &cobra.Command{
	Use:   "user",
	Short: "User management operations",
	Long:  "Create, read, update, and delete ServiceNow users",
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List users",
	Long:  "List ServiceNow users with optional filtering",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		// Parse flags
		limit, _ := cmd.Flags().GetInt("limit")
		active, _ := cmd.Flags().GetBool("active")
		department, _ := cmd.Flags().GetString("department")
		title, _ := cmd.Flags().GetString("title")
		format, _ := cmd.Flags().GetString("format")
		fields, _ := cmd.Flags().GetString("fields")

		// Create filter
		filter := &identity.UserFilter{
			Limit: limit,
		}

		if cmd.Flags().Changed("active") {
			filter.Active = &active
		}
		if department != "" {
			filter.Department = department
		}
		if title != "" {
			filter.Title = title
		}
		if fields != "" {
			filter.Fields = strings.Split(fields, ",")
		}

		// Get users
		users, err := client.Identity().ListUsers(filter)
		if err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}

		// Output results
		return outputUsers(users, format)
	},
}

var userGetCmd = &cobra.Command{
	Use:   "get [user_id_or_username]",
	Short: "Get a specific user",
	Long:  "Get detailed information about a specific user by sys_id or username",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		userID := args[0]
		format, _ := cmd.Flags().GetString("format")

		var user *identity.User

		// Try to get by sys_id first, then by username
		if len(userID) == 32 { // sys_id length
			user, err = client.Identity().GetUser(userID)
		} else {
			user, err = client.Identity().GetUserByUsername(userID)
		}

		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

		return outputUser(user, format)
	},
}

var userCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new user",
	Long:  "Create a new ServiceNow user",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		// Get user data from flags or interactive input
		userData := make(map[string]interface{})

		if username, _ := cmd.Flags().GetString("username"); username != "" {
			userData["user_name"] = username
		}
		if firstName, _ := cmd.Flags().GetString("first-name"); firstName != "" {
			userData["first_name"] = firstName
		}
		if lastName, _ := cmd.Flags().GetString("last-name"); lastName != "" {
			userData["last_name"] = lastName
		}
		if email, _ := cmd.Flags().GetString("email"); email != "" {
			userData["email"] = email
		}
		if department, _ := cmd.Flags().GetString("department"); department != "" {
			userData["department"] = department
		}

		// Create user
		user, err := client.Identity().CreateUser(userData)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		fmt.Printf("✅ Created user: %s (%s)\n", user.Name, user.UserName)
		return nil
	},
}

// Role commands
var roleCmd = &cobra.Command{
	Use:   "role",
	Short: "Role management operations",
	Long:  "Manage ServiceNow roles and role assignments",
}

var roleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List roles",
	Long:  "List ServiceNow roles with optional filtering",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")
		active, _ := cmd.Flags().GetBool("active")
		name, _ := cmd.Flags().GetString("name")
		format, _ := cmd.Flags().GetString("format")

		filter := &identity.RoleFilter{
			Limit: limit,
		}

		if cmd.Flags().Changed("active") {
			filter.Active = &active
		}
		if name != "" {
			filter.Name = name
		}

		roleClient := client.Identity().NewRoleClient()
		roles, err := roleClient.ListRoles(filter)
		if err != nil {
			return fmt.Errorf("failed to list roles: %w", err)
		}

		return outputRoles(roles, format)
	},
}

var roleAssignCmd = &cobra.Command{
	Use:   "assign [user_id] [role_name_or_id]",
	Short: "Assign a role to a user",
	Long:  "Assign a role to a user by user ID and role name or ID",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		userID := args[0]
		roleIdentifier := args[1]

		roleClient := client.Identity().NewRoleClient()

		// Get role by name or ID
		var role *identity.Role
		if len(roleIdentifier) == 32 { // sys_id length
			role, err = roleClient.GetRole(roleIdentifier)
		} else {
			role, err = roleClient.GetRoleByName(roleIdentifier)
		}

		if err != nil {
			return fmt.Errorf("failed to find role: %w", err)
		}

		// Assign role
		assignment, err := roleClient.AssignRoleToUser(userID, role.SysID)
		if err != nil {
			return fmt.Errorf("failed to assign role: %w", err)
		}

		fmt.Printf("✅ Assigned role '%s' to user %s\n", role.Name, userID)
		fmt.Printf("   Assignment ID: %s\n", assignment.SysID)
		return nil
	},
}

// Group commands
var groupCmd = &cobra.Command{
	Use:   "group",
	Short: "Group management operations",
	Long:  "Manage ServiceNow groups and group memberships",
}

var groupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List groups",
	Long:  "List ServiceNow groups with optional filtering",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")
		active, _ := cmd.Flags().GetBool("active")
		name, _ := cmd.Flags().GetString("name")
		format, _ := cmd.Flags().GetString("format")

		filter := &identity.GroupFilter{
			Limit: limit,
		}

		if cmd.Flags().Changed("active") {
			filter.Active = &active
		}
		if name != "" {
			filter.Name = name
		}

		groupClient := client.Identity().NewGroupClient()
		groups, err := groupClient.ListGroups(filter)
		if err != nil {
			return fmt.Errorf("failed to list groups: %w", err)
		}

		return outputGroups(groups, format)
	},
}

var groupAddUserCmd = &cobra.Command{
	Use:   "add-user [group_id] [user_id]",
	Short: "Add a user to a group",
	Long:  "Add a user to a ServiceNow group",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		groupID := args[0]
		userID := args[1]

		groupClient := client.Identity().NewGroupClient()
		member, err := groupClient.AddUserToGroup(userID, groupID)
		if err != nil {
			return fmt.Errorf("failed to add user to group: %w", err)
		}

		fmt.Printf("✅ Added user %s to group %s\n", userID, groupID)
		fmt.Printf("   Membership ID: %s\n", member.SysID)
		return nil
	},
}

// Access commands
var accessCmd = &cobra.Command{
	Use:   "access",
	Short: "Access control operations",
	Long:  "Check user permissions and manage access control",
}

var accessCheckCmd = &cobra.Command{
	Use:   "check [user_id] [table] [operation]",
	Short: "Check user access",
	Long:  "Check if a user has access to perform an operation on a table",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createClient()
		if err != nil {
			return err
		}

		userID := args[0]
		table := args[1]
		operation := args[2]

		accessClient := client.Identity().NewAccessClient()
		result, err := accessClient.CheckAccess(&identity.AccessCheckRequest{
			UserSysID: userID,
			Table:     table,
			Operation: operation,
		})

		if err != nil {
			return fmt.Errorf("failed to check access: %w", err)
		}

		if result.HasAccess {
			fmt.Printf("✅ User %s has %s access to %s\n", userID, operation, table)
			if result.GrantedBy != "" {
				fmt.Printf("   Granted by: %s\n", result.GrantedBy)
			}
		} else {
			fmt.Printf("❌ User %s does NOT have %s access to %s\n", userID, operation, table)
			if result.Reason != "" {
				fmt.Printf("   Reason: %s\n", result.Reason)
			}
			if len(result.RequiredRoles) > 0 {
				fmt.Printf("   Required roles: %s\n", strings.Join(result.RequiredRoles, ", "))
			}
		}

		return nil
	},
}

// Output functions
func outputUsers(users []*identity.User, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(users)
	case "table":
		fallthrough
	default:
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "SYS_ID\tUSERNAME\tNAME\tEMAIL\tACTIVE\tDEPARTMENT")
		for _, user := range users {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%t\t%s\n",
				user.SysID,
				user.UserName,
				user.Name,
				user.Email,
				user.Active,
				user.Department,
			)
		}
		return w.Flush()
	}
}

func outputUser(user *identity.User, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(user)
	default:
		fmt.Printf("User Information:\n")
		fmt.Printf("  SysID:      %s\n", user.SysID)
		fmt.Printf("  Username:   %s\n", user.UserName)
		fmt.Printf("  Name:       %s\n", user.Name)
		fmt.Printf("  Email:      %s\n", user.Email)
		fmt.Printf("  Active:     %t\n", user.Active)
		fmt.Printf("  Department: %s\n", user.Department)
		fmt.Printf("  Title:      %s\n", user.Title)
		fmt.Printf("  Manager:    %s\n", user.Manager)
		fmt.Printf("  Created:    %s\n", user.CreatedOn.Format("2006-01-02 15:04:05"))
		return nil
	}
}

func outputRoles(roles []*identity.Role, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(roles)
	default:
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "SYS_ID\tNAME\tDESCRIPTION\tACTIVE\tELEVATED")
		for _, role := range roles {
			fmt.Fprintf(w, "%s\t%s\t%s\t%t\t%t\n",
				role.SysID,
				role.Name,
				role.Description,
				role.Active,
				role.ElevatedPrivilege,
			)
		}
		return w.Flush()
	}
}

func outputGroups(groups []*identity.Group, format string) error {
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(groups)
	default:
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "SYS_ID\tNAME\tDESCRIPTION\tACTIVE\tTYPE")
		for _, group := range groups {
			fmt.Fprintf(w, "%s\t%s\t%s\t%t\t%s\n",
				group.SysID,
				group.Name,
				group.Description,
				group.Active,
				group.Type,
			)
		}
		return w.Flush()
	}
}

func init() {
	// User command flags
	userListCmd.Flags().IntP("limit", "l", 10, "Limit number of results")
	userListCmd.Flags().BoolP("active", "a", false, "Filter by active status")
	userListCmd.Flags().StringP("department", "d", "", "Filter by department")
	userListCmd.Flags().StringP("title", "t", "", "Filter by title")
	userListCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")
	userListCmd.Flags().StringP("fields", "", "", "Comma-separated list of fields to include")

	userGetCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")

	userCreateCmd.Flags().StringP("username", "u", "", "Username")
	userCreateCmd.Flags().StringP("first-name", "", "", "First name")
	userCreateCmd.Flags().StringP("last-name", "", "", "Last name")
	userCreateCmd.Flags().StringP("email", "e", "", "Email address")
	userCreateCmd.Flags().StringP("department", "d", "", "Department")

	// Role command flags
	roleListCmd.Flags().IntP("limit", "l", 10, "Limit number of results")
	roleListCmd.Flags().BoolP("active", "a", false, "Filter by active status")
	roleListCmd.Flags().StringP("name", "n", "", "Filter by role name")
	roleListCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")

	// Group command flags
	groupListCmd.Flags().IntP("limit", "l", 10, "Limit number of results")
	groupListCmd.Flags().BoolP("active", "a", false, "Filter by active status")
	groupListCmd.Flags().StringP("name", "n", "", "Filter by group name")
	groupListCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")

	// Add subcommands
	userCmd.AddCommand(userListCmd, userGetCmd, userCreateCmd)
	roleCmd.AddCommand(roleListCmd, roleAssignCmd)
	groupCmd.AddCommand(groupListCmd, groupAddUserCmd)
	accessCmd.AddCommand(accessCheckCmd)

	identityCmd.AddCommand(userCmd, roleCmd, groupCmd, accessCmd)
	rootCmd.AddCommand(identityCmd)
}