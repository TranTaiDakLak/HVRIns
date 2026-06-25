// names_default.go — Pool tên US mặc định (nhúng sẵn).
//
// Trước đây firstNames/lastNames rỗng khi user không tạo Config/Namereg/US/*.txt
// → RandomFakeProfile fallback hardcode "John"/"Smith" cho MỌI account → email toàn
// "john.smith..." (trông như bot, dễ bị flag). Nhúng pool mặc định đa dạng để có tên
// giống thật ngay từ đầu. User vẫn override được qua Config/Namereg/US/*.txt
// (ReloadOverrides chỉ ghi đè khi file có nội dung).
package fakeinfo

// defaultUSFirstNames — ~120 tên đầu phổ biến (cả nam + nữ).
var defaultUSFirstNames = []string{
	"James", "Mary", "John", "Patricia", "Robert", "Jennifer", "Michael", "Linda",
	"William", "Elizabeth", "David", "Barbara", "Richard", "Susan", "Joseph", "Jessica",
	"Thomas", "Sarah", "Charles", "Karen", "Christopher", "Nancy", "Daniel", "Lisa",
	"Matthew", "Betty", "Anthony", "Margaret", "Mark", "Sandra", "Donald", "Ashley",
	"Steven", "Kimberly", "Andrew", "Emily", "Paul", "Donna", "Joshua", "Michelle",
	"Kenneth", "Carol", "Kevin", "Amanda", "Brian", "Dorothy", "George", "Melissa",
	"Timothy", "Deborah", "Ronald", "Stephanie", "Edward", "Rebecca", "Jason", "Sharon",
	"Jeffrey", "Laura", "Ryan", "Cynthia", "Jacob", "Kathleen", "Gary", "Amy",
	"Nicholas", "Angela", "Eric", "Shirley", "Jonathan", "Anna", "Stephen", "Brenda",
	"Larry", "Pamela", "Justin", "Emma", "Scott", "Nicole", "Brandon", "Helen",
	"Benjamin", "Samantha", "Samuel", "Katherine", "Gregory", "Christine", "Alexander", "Debra",
	"Patrick", "Rachel", "Frank", "Carolyn", "Raymond", "Janet", "Jack", "Maria",
	"Dennis", "Olivia", "Jerry", "Heather", "Tyler", "Diane", "Aaron", "Julie",
	"Jose", "Joyce", "Adam", "Victoria", "Nathan", "Kelly", "Henry", "Christina",
	"Zachary", "Lauren", "Douglas", "Joan", "Peter", "Evelyn", "Kyle", "Hannah",
}

// defaultUSLastNames — ~120 họ phổ biến.
var defaultUSLastNames = []string{
	"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis",
	"Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson", "Thomas",
	"Taylor", "Moore", "Jackson", "Martin", "Lee", "Perez", "Thompson", "White",
	"Harris", "Sanchez", "Clark", "Ramirez", "Lewis", "Robinson", "Walker", "Young",
	"Allen", "King", "Wright", "Scott", "Torres", "Nguyen", "Hill", "Flores",
	"Green", "Adams", "Nelson", "Baker", "Hall", "Rivera", "Campbell", "Mitchell",
	"Carter", "Roberts", "Gomez", "Phillips", "Evans", "Turner", "Diaz", "Parker",
	"Cruz", "Edwards", "Collins", "Reyes", "Stewart", "Morris", "Morales", "Murphy",
	"Cook", "Rogers", "Gutierrez", "Ortiz", "Morgan", "Cooper", "Peterson", "Bailey",
	"Reed", "Kelly", "Howard", "Ramos", "Kim", "Cox", "Ward", "Richardson",
	"Watson", "Brooks", "Chavez", "Wood", "James", "Bennett", "Gray", "Mendoza",
	"Ruiz", "Hughes", "Price", "Alvarez", "Castillo", "Sanders", "Patel", "Myers",
	"Long", "Ross", "Foster", "Jimenez", "Powell", "Jenkins", "Perry", "Russell",
	"Sullivan", "Bell", "Coleman", "Butler", "Henderson", "Barnes", "Gonzales", "Fisher",
	"Vasquez", "Simmons", "Romero", "Jordan", "Patterson", "Alexander", "Hamilton", "Graham",
}
