//go:generate go-enum --names --values -t ./enum_i18n.tmpl

package model

// Activity Action
/*
ENUM(
   register, // Register
   register_verify, // Registration verification
   login, // Login
   logout, // Logout
   update_intro, // Update introduction
   create_article, // Create article
   reply_article, // Reply to article
   edit_article, // Edit article
   delete_article, // Delete article
   save_article, // Save article
   vote_article, // Vote article
   react_article, // React to article
   set_role, // Set role
   add_role, // Add role
   edit_role, // Edit role
   subscribe_article, // Subscribe article
   retrieve_password, // Retrieve password
   reset_password, // Reset password
   toggle_hide_history, // Toggle hide history
   recover, // Recover article
   block_regions, // Block regions
   lock_article, // Lock article
   fade_out_article, // Fade out article
)
*/
type AcAction string
