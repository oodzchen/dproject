//go:generate go-enum --names --values -t ./enum_text.tmpl

package model

// Activity Action
/*
ENUM(
   register, // Register
   login, // Login
   logout, // Logout
   update_intro, // Update introduction
   create_article, // Create article
   edit_article, // Edit article
   delete_article, // Delete article
   save_article, // Save article
   vote_article, // Vote article
   react_article, // React to article
   set_role, // Set role
   add_role, // Add role
   edit_role, // Edit role
)
*/
type AcAction string
