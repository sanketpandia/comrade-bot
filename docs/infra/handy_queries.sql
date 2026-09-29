SELECT u.if_community_id, vau.role, from va_user_roles vau
JOIN users u on u.id = vau.user_id
JOIN virtual_airlines va ON va.id = vau.va_id;